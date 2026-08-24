package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	ivdomain "github.com/intivai/backend/internal/interview/domain"
	"github.com/intivai/backend/pkg/db"
	"gorm.io/gorm"
)

// interviewColumns is the column list for the interviews table. GetByID and
// ByApplication use it unqualified; ListByOrg qualifies it with the iv. alias
// for the applications join.
const interviewColumns = `id, application_id, status, transcript, last_question_idx, context_version, evaluation, consent_given, human_requested, proctoring_events, proctoring_summary, coding_sessions, started_at, completed_at, expires_at, created_at`

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
	raw, rawEvents, rawSummary, rawSessions, err := marshalInterview(iv)
	if err != nil {
		return err
	}
	return db.WrapError(tx.WithContext(ctx).Exec(
		`INSERT INTO interviews (id, application_id, type, status, transcript, last_question_idx,
		 context_version, proctoring_events, proctoring_summary, coding_sessions,
		 started_at, completed_at, expires_at, created_at)
		 VALUES ($1, $2, 'chat', $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		iv.ID, iv.ApplicationID, string(iv.Status), raw, iv.LastQuestionIdx,
		iv.ContextVersion, rawEvents, rawSummary, rawSessions,
		iv.StartedAt, iv.CompletedAt, iv.ExpiresAt, iv.CreatedAt).Error)
}

// GetByID hydrates OrgID through the applications join (interviews have no
// org_id column; RLS resolves via applications). Domain callers that compare
// iv.OrgID (telemetry ticket check, recruiter decision ownership) depend on
// it — keep the join.
func (r *PostgresInterviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*ivdomain.Interview, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	row := tx.Raw(
		`SELECT iv.`+strings.ReplaceAll(interviewColumns, ", ", ", iv.")+`, a.org_id
		 FROM interviews iv
		 JOIN applications a ON a.id = iv.application_id
		 WHERE iv.id = $1`, id).Row()
	return scanInterview(row)
}

func (r *PostgresInterviewRepo) Update(ctx context.Context, iv *ivdomain.Interview) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	raw, rawEvents, rawSummary, rawSessions, err := marshalInterview(iv)
	if err != nil {
		return err
	}
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
	if err := decodeJSONB(rawEvents, &events); err != nil {
		return fmt.Errorf("decode proctoring events: %w", err)
	}
	events = append(events, event)
	// The summary reflects the FULL event history (dropped raw events must
	// not silently weaken the integrity score)…
	summary, err := json.Marshal(ivdomain.CalculateProctoringSummary(events))
	if err != nil {
		return fmt.Errorf("encode proctoring summary: %w", err)
	}
	// …but raw events are retention-capped (design decision): keep the most
	// recent 500 so the JSONB column cannot grow unboundedly per interview.
	const maxStoredEvents = 500
	if len(events) > maxStoredEvents {
		events = events[len(events)-maxStoredEvents:]
	}
	raw, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("encode proctoring events: %w", err)
	}
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
	raw, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("encode coding session: %w", err)
	}
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

func (r *PostgresInterviewRepo) SetHumanRequested(ctx context.Context, id uuid.UUID, requested bool) error {
	tx, err := r.tx(ctx)
	if err != nil {
		return err
	}
	return db.WrapError(tx.WithContext(ctx).Exec(
		`UPDATE interviews SET human_requested = $1, updated_at = NOW() WHERE id = $2`, requested, id).Error)
}

func (r *PostgresInterviewRepo) ByApplication(ctx context.Context, applicationID uuid.UUID) ([]*ivdomain.Interview, error) {
	tx, err := r.tx(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Raw(
		`SELECT iv.`+strings.ReplaceAll(interviewColumns, ", ", ", iv.")+`, a.org_id
		 FROM interviews iv
		 JOIN applications a ON a.id = iv.application_id
		 WHERE iv.application_id = $1 ORDER BY iv.created_at DESC`, applicationID).Rows()
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
		`SELECT iv.`+strings.ReplaceAll(interviewColumns, ", ", ", iv.")+`, a.org_id
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
func marshalInterview(iv *ivdomain.Interview) (rawTranscript, rawEvents, rawSummary, rawSessions []byte, err error) {
	rawTranscript, err = json.Marshal(transcript{Questions: iv.Questions, Answers: iv.Answers})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("encode transcript: %w", err)
	}
	rawEvents, err = json.Marshal(iv.ProctoringEvents)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("encode proctoring events: %w", err)
	}
	rawSummary, err = json.Marshal(iv.ProctoringSummary)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("encode proctoring summary: %w", err)
	}
	rawSessions, err = json.Marshal(iv.CodingSessions)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("encode coding sessions: %w", err)
	}
	return rawTranscript, rawEvents, rawSummary, rawSessions, nil
}

func decodeJSONB[T any](raw []byte, dst *T) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return json.Unmarshal(raw, dst)
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
		&iv.ContextVersion, &iv.Evaluation, &iv.ConsentGiven, &iv.HumanRequested, &rawEvents, &rawSummary, &rawSessions,
		&iv.StartedAt, &iv.CompletedAt, &iv.ExpiresAt, &iv.CreatedAt, &iv.OrgID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ivdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var t transcript
	if err := decodeJSONB(rawTranscript, &t); err != nil {
		return nil, fmt.Errorf("decode transcript: %w", err)
	}
	if err := decodeJSONB(rawEvents, &iv.ProctoringEvents); err != nil {
		return nil, fmt.Errorf("decode proctoring events: %w", err)
	}
	if err := decodeJSONB(rawSummary, &iv.ProctoringSummary); err != nil {
		return nil, fmt.Errorf("decode proctoring summary: %w", err)
	}
	if err := decodeJSONB(rawSessions, &iv.CodingSessions); err != nil {
		return nil, fmt.Errorf("decode coding sessions: %w", err)
	}
	iv.Questions = t.Questions
	iv.Answers = t.Answers

	iv.SetClock(ivdomain.SystemClock())
	return &iv, nil
}
