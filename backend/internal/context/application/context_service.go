package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	ctxdomain "github.com/intivai/backend/internal/context/domain"
	"github.com/intivai/backend/internal/iam/application"
	iamdomain "github.com/intivai/backend/internal/iam/domain"
	memdomain "github.com/intivai/backend/internal/memory/domain"
	sharederr "github.com/intivai/backend/internal/shared/errors"
	"github.com/intivai/backend/internal/shared/uuidx"
	"github.com/intivai/backend/pkg/db"
	"github.com/intivai/backend/pkg/queue"
	"github.com/intivai/backend/pkg/storage"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

const (
	TaskIndexContext = "index_context"
	ctxMaxBytes      = 64 * 1024
)

// ContextService: company context upload with hash dedup + versioning,
// tenant prompt set/get with injection validation.
type ContextService struct {
	pool  *gorm.DB
	repo  ctxdomain.ContextRepository
	store *storage.Storage
	queue *queue.Client
	log   zerolog.Logger
}

func NewContextService(pool *gorm.DB, repo ctxdomain.ContextRepository, store *storage.Storage, queueClient *queue.Client, log zerolog.Logger) *ContextService {
	return &ContextService{pool: pool, repo: repo, store: store, queue: queueClient, log: log}
}

type ContextResult struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	Version     int       `json:"version"`
	ContentHash string    `json:"content_hash"`
	CreatedAt   string    `json:"created_at"`
}

// UploadContext stores file/text content, dedups by sha256, bumps version,
// then enqueues index_context.
func (s *ContextService) UploadContext(ctx context.Context, actor application.AuthContext, contentType string, content []byte) (*ContextResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	if len(content) == 0 {
		return nil, sharederr.NewDomainError("CTX_EMPTY", "context content is empty")
	}
	if len(content) > ctxMaxBytes {
		return nil, sharederr.NewDomainError("CTX_TOO_LARGE", "context content exceeds 64KB")
	}
	if ctxdomain.ContainsInjection(string(content)) {
		return nil, sharederr.NewDomainError("CTX_INJECTION", "context content contains forbidden content")
	}
	hash := sha256.Sum256(content)
	hashHex := hex.EncodeToString(hash[:])

	storagePath := fmt.Sprintf("contexts/%s/%s.%s", actor.OrgID, hashHex, ext(contentType))

	var cc *ctxdomain.CompanyContext
	created := false
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		existing, err := s.repo.GetContextByHash(tctx, actor.OrgID, hashHex)
		if err == nil && existing != nil {
			cc = existing
			return nil
		}
		if err != ctxdomain.ErrNotFound {
			return err
		}
		latest, err := s.repo.ListContexts(tctx, actor.OrgID)
		if err != nil {
			return err
		}
		version := 1
		if len(latest) > 0 {
			version = latest[0].Version + 1
		}
		cc, err = ctxdomain.NewCompanyContext(actor.OrgID, contentType, hashHex, storagePath)
		if err != nil {
			return err
		}
		cc.Version = version
		if err := s.repo.CreateContext(tctx, cc); err != nil {
			return err
		}
		created = true
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Upload AFTER the tx: a slow object-store write must not hold a DB
	// connection (pool starvation), and the row only exists if we get here.
	if created {
		if err := s.store.Upload(ctx, storagePath, strings.NewReader(string(content)), int64(len(content)), mimeOf(contentType)); err != nil {
			// Orphaned row without the object — best-effort cleanup + fail.
			_ = db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
				return s.repo.DeleteContext(tctx, actor.OrgID, cc.ID)
			})
			return nil, err
		}
	}

	if created {
		if _, err := s.queue.Enqueue(ctx, TaskIndexContext, IndexContextPayload{
			OrgID: actor.OrgID.String(), ContextID: cc.ID.String(),
		}); err != nil {
			s.log.Error().Err(err).Msg("enqueue index_context failed")
		}
	}
	return &ContextResult{ID: cc.ID, Type: cc.Type, Version: cc.Version, ContentHash: cc.ContentHash, CreatedAt: cc.CreatedAt.String()}, nil
}

type SetPromptCommand struct {
	SystemPrompt string
}

type PromptResult struct {
	SystemPrompt string `json:"system_prompt"`
	Version      int    `json:"version"`
}

func (s *ContextService) SetPrompt(ctx context.Context, actor application.AuthContext, cmd SetPromptCommand) (*PromptResult, error) {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return nil, err
	}
	if err := ctxdomain.ValidatePrompt(cmd.SystemPrompt); err != nil {
		return nil, err
	}
	var result *PromptResult
	err := db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		// Removed lockOrg to avoid long locks
		version := 1
		if latest, err := s.repo.GetLatestPrompt(tctx, actor.OrgID); err == nil {
			version = latest.Version + 1
		} else if err != ctxdomain.ErrNotFound {
			return err
		}
		p := &ctxdomain.TenantPrompt{OrgID: actor.OrgID, SystemPrompt: cmd.SystemPrompt, Version: version}
		if err := s.repo.SetPrompt(tctx, p); err != nil {
			return err
		}
		result = &PromptResult{SystemPrompt: p.SystemPrompt, Version: p.Version}
		return nil
	})
	return result, err
}

// GetPrompt falls back to the global default when the tenant has none.
func (s *ContextService) GetPrompt(ctx context.Context, actor application.AuthContext) (*PromptResult, error) {
	p, err := s.repo.GetLatestPrompt(ctx, actor.OrgID)
	if errors.Is(err, ctxdomain.ErrNotFound) {
		return &PromptResult{SystemPrompt: ctxdomain.DefaultPrompt(), Version: 0}, nil
	}
	if err != nil {
		return nil, err
	}
	return &PromptResult{SystemPrompt: p.SystemPrompt, Version: p.Version}, nil
}

func (s *ContextService) ListContexts(ctx context.Context, actor application.AuthContext) ([]*ContextResult, error) {
	list, err := s.repo.ListContexts(ctx, actor.OrgID)
	if err != nil {
		return nil, err
	}
	out := make([]*ContextResult, 0, len(list))
	for _, cc := range list {
		out = append(out, &ContextResult{ID: cc.ID, Type: cc.Type, Version: cc.Version, ContentHash: cc.ContentHash, CreatedAt: cc.CreatedAt.String()})
	}
	return out, nil
}

// Delete removes one company-context row for the org. The stored object is
// intentionally kept: objects are content-hash deduped per org, so the object
// may back another version/row, and deleting it could orphan future lookups.
func (s *ContextService) Delete(ctx context.Context, actor application.AuthContext, id uuid.UUID) error {
	if err := application.Authorize(actor, iamdomain.RoleAdmin, iamdomain.RoleRecruiter); err != nil {
		return err
	}
	return db.RunInTx(ctx, s.pool, actor.OrgID.String(), func(tctx context.Context) error {
		// RLS scopes the read to the actor's org; ErrNotFound → 404.
		if _, err := s.repo.GetContextByID(tctx, id); err != nil {
			if errors.Is(err, ctxdomain.ErrNotFound) {
				return sharederr.NewNotFoundError("context", id.String())
			}
			return err
		}
		return s.repo.DeleteContext(tctx, actor.OrgID, id)
	})
}

// IndexWorker: index context content into the tenant's Mnemosyne bank.
type IndexWorker struct {
	pool   *gorm.DB
	repo   ctxdomain.ContextRepository
	store  *storage.Storage
	memory memdomain.BankFactory
	log    zerolog.Logger
}

type IndexContextPayload struct {
	OrgID     string `json:"org_id"`
	ContextID string `json:"context_id"`
}

func NewIndexWorker(pool *gorm.DB, repo ctxdomain.ContextRepository, store *storage.Storage, memory memdomain.BankFactory, log zerolog.Logger) *IndexWorker {
	return &IndexWorker{pool: pool, repo: repo, store: store, memory: memory, log: log}
}

func (w *IndexWorker) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskIndexContext, w.handle)
}

func (w *IndexWorker) handle(ctx context.Context, t *asynq.Task) error {
	var p IndexContextPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return asynq.SkipRetry
	}
	ctxID := uuidx.MustParse(p.ContextID)

	var cc *ctxdomain.CompanyContext
	err := db.RunInTx(ctx, w.pool, p.OrgID, func(tctx context.Context) error {
		var err error
		cc, err = w.repo.GetContextByID(tctx, ctxID)
		return err
	})
	if err != nil {
		return asynq.SkipRetry
	}

	content, err := w.store.Download(ctx, cc.StoragePath)
	if err != nil {
		w.log.Error().Err(err).Msg("index_context download failed")
		return err
	}
	defer content.Close()
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, content); err != nil {
		return err
	}
	return w.memory.ForBank(p.OrgID).Remember(ctx, "company_context_"+cc.Type, buf.String(), 0.8)
}

func ext(contentType string) string {
	if contentType == ctxdomain.TypeFile {
		return "bin"
	}
	return "txt"
}

func mimeOf(contentType string) string {
	if contentType == ctxdomain.TypeFile {
		return "application/octet-stream"
	}
	return "text/plain"
}
