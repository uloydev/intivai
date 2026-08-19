package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// interviewColumns is the column list for the interviews table. GetByID and
// ByApplication use it unqualified; ListByOrg qualifies it with the iv. alias
// for the applications join.
const interviewColumns = `id, application_id, status, transcript, last_question_idx, context_version, evaluation, consent_given, proctoring_events, proctoring_summary, coding_sessions, started_at, completed_at, expires_at, created_at`

type PostgresInterviewRepo struct {
	pool *gorm.DB
}

func NewPostgresInterviewRepo(pool *gorm.DB) *PostgresInterviewRepo {
	return &PostgresInterviewRepo{pool: pool}
}

func (r *PostgresInterviewRepo) tx(ctx context.Context) (*gorm.DB, error) {
	tx, ok := db.TxFrom(ctx)
	if !ok {
		return nil, db.ErrNoTx
	}
	return tx, nil
}

func (r *PostgresInterviewRepo) Create(ctx context.Context, iv *ivdomain.Interview) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	raw, rawEvents, rawSummary, rawSessions := marshalInterview(iv)
	return db.WrapError(tx.WithContext(ctx).Exec(
		`INSERT INTO interviews (id, application_id, type, status, transcript, last_question_idx,
		 context_version, proctoring_events, proctoring_summary, coding_sessions,
		 started_at, completed_at, expires_at, created_at)
		 VALUES ($1, $2, 'chat', $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		iv.ID, iv.ApplicationID, string(iv.Status), raw, iv.LastQuestionIdx,
		iv.ContextVersion, rawEvents, rawSummary, rawSessions,
		iv.StartedAt, iv.CompletedAt, iv.ExpiresAt, iv.CreatedAt).Error)
}

func (r *PostgresInterviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*ivdomain.Interview, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT `+interviewColumns+` FROM interviews WHERE id = $1`, id).Row()
	return scanInterview(row)
}

func (r *PostgresInterviewRepo) Update(ctx context.Context, iv *ivdomain.Interview) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	raw, rawEvents, rawSummary, rawSessions := marshalInterview(iv)
	return db.WrapError(tx.WithContext(ctx).Exec(
		`UPDATE interviews SET status = $1, transcript = $2, last_question_idx = $3,
		 proctoring_events = $4, proctoring_summary = $5, coding_sessions = $6,
		 started_at = $7, completed_at = $8, expires_at = $9, updated_at = NOW() WHERE id = $10`,
		string(iv.Status), raw, iv.LastQuestionIdx, rawEvents, rawSummary, rawSessions,
		iv.StartedAt, iv.CompletedAt, iv.ExpiresAt, iv.ID).Error)
}

// RecordProctoringEvent appends to the events JSONB and recomputes the
// summary — column-scoped (reads/writes only proctoring_*), so it never
// races the transcript the way a full read-modify-write Update() would
// (keystroke Touch() and answer commits run concurrently).
func (r *PostgresInterviewRepo) RecordProctoringEvent(ctx context.Context, id uuid.UUID, event ivdomain.ProctoringEvent) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	var rawEvents []byte
	row := tx.Raw(
		`SELECT COALESCE(proctoring_events, '[]'::jsonb) FROM interviews WHERE id = $1`, id).Row()
	if err := row.Scan(&rawEvents); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ivdomain.ErrNotFound
		}
		return err
	}
	var events []ivdomain.ProctoringEvent
	_ = json.Unmarshal(rawEvents, &events)
	events = append(events, event)
	// The summary reflects the FULL event history (dropped raw events must
	// not silently weaken the integrity score)…
	summary, _ := json.Marshal(ivdomain.CalculateProctoringSummary(events))
	// …but raw events are retention-capped (design decision): keep the most
	// recent 500 so the JSONB column cannot grow unboundedly per interview.
	const maxStoredEvents = 500
	if len(events) > maxStoredEvents {
		events = events[len(events)-maxStoredEvents:]
	}
	raw, _ := json.Marshal(events)
	return tx.WithContext(ctx).Exec(
		`UPDATE interviews SET
		   proctoring_events = $1,
		   proctoring_summary = $2,
		   updated_at = NOW()
		 WHERE id = $3`,
		string(raw), string(summary), id).Error
}

// Touch refreshes updated_at + expires_at only — never rewrites transcript.
func (r *PostgresInterviewRepo) Touch(ctx context.Context, id uuid.UUID) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`UPDATE interviews SET updated_at = NOW() WHERE id = $1`, id).Error
}

// RecordCodingSession appends one snapshot to coding_sessions JSONB.
func (r *PostgresInterviewRepo) RecordCodingSession(ctx context.Context, id uuid.UUID, session ivdomain.CodingSession) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(session)
	return tx.WithContext(ctx).Exec(
		`UPDATE interviews SET
		   coding_sessions = COALESCE(coding_sessions, '[]'::jsonb) || $1::jsonb,
		   updated_at = NOW()
		 WHERE id = $2`,
		string(raw), id).Error
}

// SaveEvaluation persists the report, but NEVER overwrites an existing one —
// the inline WS evaluation and the async worker can race; first writer wins
// (atomic WHERE guard, not read-then-write).
func (r *PostgresInterviewRepo) SaveEvaluation(ctx context.Context, id uuid.UUID, report []byte) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	res := tx.WithContext(ctx).Exec(
		`UPDATE interviews SET evaluation = $1, updated_at = NOW() WHERE id = $2 AND evaluation IS NULL`,
		report, id)
	if res.Error != nil {
		return db.WrapError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ivdomain.ErrEvaluationExists
	}
	return nil
}

func (r *PostgresInterviewRepo) SetConsent(ctx context.Context, id uuid.UUID) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return db.WrapError(tx.WithContext(ctx).Exec(
		`UPDATE interviews SET consent_given = true, updated_at = NOW() WHERE id = $1`, id).Error)
}

func (r *PostgresInterviewRepo) ByApplication(ctx context.Context, applicationID uuid.UUID) ([]*ivdomain.Interview, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Raw(
		`SELECT `+interviewColumns+` FROM interviews WHERE application_id = $1 ORDER BY created_at DESC`, applicationID).Rows()
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
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Raw(
		`SELECT iv.`+strings.ReplaceAll(interviewColumns, ", ", ", iv.")+`
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

// marshalInterview encodes the JSONB columns shared by Create and Update
// (transcript, proctoring events/summary, coding sessions).
func marshalInterview(iv *ivdomain.Interview) (rawTranscript, rawEvents, rawSummary, rawSessions []byte) {
	raw, _ := json.Marshal(transcript{Questions: iv.Questions, Answers: iv.Answers})
	rawEvents, _ = json.Marshal(iv.ProctoringEvents)
	rawSummary, _ = json.Marshal(iv.ProctoringSummary)
	rawSessions, _ = json.Marshal(iv.CodingSessions)
	return raw, rawEvents, rawSummary, rawSessions
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInterview(row rowScanner) (*ivdomain.Interview, error) {
	var (
		iv            ivdomain.Interview
		rawTranscript []byte
		rawEvents     []byte
		rawSummary    []byte
		rawSessions   []byte
	)
	err := row.Scan(&iv.ID, &iv.ApplicationID, &iv.Status, &rawTranscript, &iv.LastQuestionIdx,
		&iv.ContextVersion, &iv.Evaluation, &iv.ConsentGiven, &rawEvents, &rawSummary, &rawSessions,
		&iv.StartedAt, &iv.CompletedAt, &iv.ExpiresAt, &iv.CreatedAt)
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
	if len(rawEvents) > 0 {
		_ = json.Unmarshal(rawEvents, &iv.ProctoringEvents)
	}
	if len(rawSummary) > 0 {
		_ = json.Unmarshal(rawSummary, &iv.ProctoringSummary)
	}
	if len(rawSessions) > 0 {
		_ = json.Unmarshal(rawSessions, &iv.CodingSessions)
	}

	iv.SetClock(ivdomain.SystemClock())
	return &iv, nil
}
