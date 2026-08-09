package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

type PostgresInterviewRepo struct {
	pool *gorm.DB
}

func NewPostgresInterviewRepo(pool *gorm.DB) *PostgresInterviewRepo {
	return &PostgresInterviewRepo{pool: pool}
}

func (r *PostgresInterviewRepo) q(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresInterviewRepo) Create(ctx context.Context, iv *ivdomain.Interview) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(transcript{Questions: iv.Questions, Answers: iv.Answers})
	return q.WithContext(ctx).Exec(
		`INSERT INTO interviews (id, application_id, type, status, transcript, last_question_idx,
		 context_version, started_at, completed_at, expires_at, created_at)
		 VALUES ($1, $2, 'chat', $3, $4, $5, $6, $7, $8, $9, $10)`,
		iv.ID, iv.ApplicationID, string(iv.Status), raw, iv.LastQuestionIdx,
		iv.ContextVersion, iv.StartedAt, iv.CompletedAt, iv.ExpiresAt, iv.CreatedAt).Error
}

func (r *PostgresInterviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*ivdomain.Interview, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	row := q.Raw(
		`SELECT id, application_id, status, transcript, last_question_idx, context_version,
		 evaluation, consent_given, started_at, completed_at, expires_at, created_at
		 FROM interviews WHERE id = $1`, id).Row()
	return scanInterview(row)
}

func (r *PostgresInterviewRepo) Update(ctx context.Context, iv *ivdomain.Interview) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(transcript{Questions: iv.Questions, Answers: iv.Answers})
	return q.WithContext(ctx).Exec(
		`UPDATE interviews SET status = $1, transcript = $2, last_question_idx = $3,
		 started_at = $4, completed_at = $5, expires_at = $6, updated_at = NOW() WHERE id = $7`,
		string(iv.Status), raw, iv.LastQuestionIdx, iv.StartedAt, iv.CompletedAt, iv.ExpiresAt, iv.ID).Error
}

// SaveEvaluation persists the report, but NEVER overwrites an existing one —
// the inline WS evaluation and the async worker can race; first writer wins
// (atomic WHERE guard, not read-then-write).
func (r *PostgresInterviewRepo) SaveEvaluation(ctx context.Context, id uuid.UUID, report []byte) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	res := q.WithContext(ctx).Exec(
		`UPDATE interviews SET evaluation = $1, updated_at = NOW() WHERE id = $2 AND evaluation IS NULL`,
		report, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ivdomain.ErrEvaluationExists
	}
	return nil
}

func (r *PostgresInterviewRepo) SetConsent(ctx context.Context, id uuid.UUID) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE interviews SET consent_given = true, updated_at = NOW() WHERE id = $1`, id).Error
}

func (r *PostgresInterviewRepo) ByApplication(ctx context.Context, applicationID uuid.UUID) ([]*ivdomain.Interview, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.Raw(
		`SELECT id, application_id, status, transcript, last_question_idx, context_version,
		 evaluation, consent_given, started_at, completed_at, expires_at, created_at
		 FROM interviews WHERE application_id = $1 ORDER BY created_at DESC`, applicationID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*ivdomain.Interview{}
	for rows.Next() {
		iv, err := scanInterview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, iv)
	}
	return out, rows.Err()
}

// ListByOrg — interviews of one org, newest first. RLS applies through the
// applications join (interviews have no org_id column).
func (r *PostgresInterviewRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*ivdomain.Interview, error) {
	q, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := q.Raw(
		`SELECT iv.id, iv.application_id, iv.status, iv.transcript, iv.last_question_idx,
		 iv.context_version, iv.evaluation, iv.consent_given, iv.started_at, iv.completed_at,
		 iv.expires_at, iv.created_at
		 FROM interviews iv
		 JOIN applications a ON a.id = iv.application_id
		 WHERE a.org_id = $1
		 ORDER BY iv.created_at DESC`, orgID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*ivdomain.Interview{}
	for rows.Next() {
		iv, err := scanInterview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, iv)
	}
	return out, rows.Err()
}

type transcript struct {
	Questions []ivdomain.Question `json:"questions"`
	Answers   []ivdomain.Answer   `json:"answers"`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInterview(row rowScanner) (*ivdomain.Interview, error) {
	var (
		iv            ivdomain.Interview
		rawTranscript []byte
	)
	err := row.Scan(&iv.ID, &iv.ApplicationID, &iv.Status, &rawTranscript, &iv.LastQuestionIdx,
		&iv.ContextVersion, &iv.Evaluation, &iv.ConsentGiven, &iv.StartedAt, &iv.CompletedAt, &iv.ExpiresAt, &iv.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ivdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(rawTranscript) > 0 {
		var t transcript
		if err := json.Unmarshal(rawTranscript, &t); err == nil {
			iv.Questions = t.Questions
			iv.Answers = t.Answers
		}
	}
	iv.SetClock(ivdomain.SystemClock())
	return &iv, nil
}
