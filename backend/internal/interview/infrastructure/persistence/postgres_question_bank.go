package persistence

import (
	"context"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type PostgresQuestionBank struct {
	pool *gorm.DB
}

func NewPostgresQuestionBank(pool *gorm.DB) *PostgresQuestionBank {
	return &PostgresQuestionBank{pool: pool}
}

func (r *PostgresQuestionBank) q(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresQuestionBank) Create(ctx context.Context, orgID uuid.UUID, q ivdomain.Question) error {
	qx, err := r.q(ctx)
	if err != nil {
		return err
	}
	skills := []string{}
	if q.Skill != "" {
		skills = []string{q.Skill}
	}
	return qx.WithContext(ctx).Exec(
		`INSERT INTO questions (id, org_id, category, difficulty, body, skills, created_at)
		 VALUES ($1, $2, $3, 'medium', $4, $5, NOW())`,
		uuid.New(), orgID, q.Category, q.Content, pq.Array(skills)).Error
}

func (r *PostgresQuestionBank) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]ivdomain.Question, error) {
	qx, err := r.q(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := qx.Raw(
		`SELECT category, body, skills FROM questions WHERE org_id = $1 ORDER BY created_at DESC LIMIT 100`,
		orgID).Rows()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []ivdomain.Question{}
	for rows.Next() {
		var (
			q      ivdomain.Question
			skills []string
		)
		if err := rows.Scan(&q.Category, &q.Content, pq.Array(&skills)); err != nil {
			return nil, err
		}
		if len(skills) > 0 {
			q.Skill = skills[0]
		}
		out = append(out, q)
	}
	return out, rows.Err()
}
