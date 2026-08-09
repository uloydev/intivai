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
		 evaluation, started_at, completed_at, expires_at, created_at
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

func (r *PostgresInterviewRepo) SaveEvaluation(ctx context.Context, id uuid.UUID, report []byte) error {
	q, err := r.q(ctx)
	if err != nil {
		return err
	}
	return q.WithContext(ctx).Exec(
		`UPDATE interviews SET evaluation = $1, updated_at = NOW() WHERE id = $2`,
		report, id).Error
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
		&iv.ContextVersion, &iv.Evaluation, &iv.StartedAt, &iv.CompletedAt, &iv.ExpiresAt, &iv.CreatedAt)
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
