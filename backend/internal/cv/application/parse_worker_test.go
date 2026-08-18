package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

func TestParseWorkerBadPayload(t *testing.T) {
	worker := &ParseWorker{}
	task := asynq.NewTask(TaskParseCV, []byte("invalid json"))
	err := worker.handle(context.Background(), task)
	if err != asynq.SkipRetry {
		t.Fatalf("expected asynq.SkipRetry for bad payload, got %v", err)
	}

	badUUIDPayload, _ := json.Marshal(ParseCVPayload{
		OrgID:       uuid.New().String(),
		CandidateID: "not-a-uuid",
	})
	task2 := asynq.NewTask(TaskParseCV, badUUIDPayload)
	err2 := worker.handle(context.Background(), task2)
	if err2 != asynq.SkipRetry {
		t.Fatalf("expected asynq.SkipRetry for invalid candidate UUID, got %v", err2)
	}
}

func TestExtractPDFTextEmpty(t *testing.T) {
	_, err := extractPDFText([]byte("not a valid pdf header"))
	if err == nil {
		t.Fatal("expected error on invalid pdf bytes")
	}
}
