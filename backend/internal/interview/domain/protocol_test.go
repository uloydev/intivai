package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestServerMessagesRoundTrip(t *testing.T) {
	start := NewInterviewStart("iv_abc", 5)
	raw, err := json.Marshal(start)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != MsgStart || got["session_id"] != "iv_abc" || got["total_questions"] != float64(5) {
		t.Fatalf("start message: %v", got)
	}
}

func TestParseClientMessageValid(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{`{"type":"answer","content":"In my previous role...","idx":1}`, MsgAnswer},
		{`{"type":"interrupt"}`, MsgInterrupt},
		{`{"type":"ping"}`, MsgPing},
		{`{"type":"resume","session_id":"iv_abc"}`, MsgResume},
		{`{"type":"candidate_question","content":"Does this role offer remote work?"}`, MsgCandidateQuestion},
	}
	for _, c := range cases {
		msg, err := ParseClientMessage([]byte(c.raw))
		if err != nil {
			t.Fatalf("%s: %v", c.want, err)
		}
		switch m := msg.(type) {
		case AnswerMessage:
			if c.want != MsgAnswer || m.Idx != 1 || m.Content == "" {
				t.Fatalf("answer mismatch: %+v", m)
			}
		case InterruptMessage, PingMessage, ResumeMessage:
			// typed correctly — nothing more to assert
		case CandidateQuestionMessage:
			if c.want != MsgCandidateQuestion || m.Content == "" {
				t.Fatalf("candidate question mismatch: %+v", m)
			}
		default:
			t.Fatalf("unexpected type %T", msg)
		}
	}
}

func TestParseClientMessageRejectsUnknownAndMalformed(t *testing.T) {
	if _, err := ParseClientMessage([]byte(`{"type":"hack","x":1}`)); err == nil {
		t.Fatal("unknown type accepted")
	}
	if _, err := ParseClientMessage([]byte(`not json`)); err == nil {
		t.Fatal("malformed frame accepted")
	}
	if _, err := ParseClientMessage([]byte(`{"type":"answer"}`)); err == nil {
		t.Fatal("answer without content accepted")
	}
	if _, err := ParseClientMessage([]byte(`{"type":"candidate_question"}`)); err == nil {
		t.Fatal("candidate question without content accepted")
	}
}

// G11: answer text is client-controlled and fed to the LLM — an overlength
// frame must be rejected at parse time, aligned with the service clamp (4000).
func TestParseAnswerContentCap(t *testing.T) {
	atCap := strings.Repeat("x", 4000)
	if _, err := ParseClientMessage([]byte(`{"type":"answer","content":"` + atCap + `","idx":1}`)); err != nil {
		t.Fatalf("answer at cap rejected: %v", err)
	}
	over := strings.Repeat("x", 4001)
	if _, err := ParseClientMessage([]byte(`{"type":"answer","content":"` + over + `","idx":1}`)); err == nil {
		t.Fatal("overlength answer accepted")
	}
}

// D12/I13: candidate question text is client-controlled and persisted to
// qa_pairs — an overlength frame must be rejected at parse time.
func TestParseCandidateQuestionContentCap(t *testing.T) {
	atCap := strings.Repeat("x", 1000)
	if _, err := ParseClientMessage([]byte(`{"type":"candidate_question","content":"` + atCap + `"}`)); err != nil {
		t.Fatalf("content at cap rejected: %v", err)
	}
	over := strings.Repeat("x", 1001)
	if _, err := ParseClientMessage([]byte(`{"type":"candidate_question","content":"` + over + `"}`)); err == nil {
		t.Fatal("overlength candidate question accepted")
	}
}

func TestQAAnswerMessageRoundTrip(t *testing.T) {
	ans := QAAnswerMessage{Type: MsgQaAnswer, Question: "Is relocation required?", Answer: "This role is fully remote.", Refused: false}
	raw, err := json.Marshal(ans)
	if err != nil {
		t.Fatal(err)
	}
	var got QAAnswerMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Type != MsgQaAnswer || got.Question != ans.Question || got.Answer != ans.Answer || got.Refused {
		t.Fatalf("qa answer mismatch: %+v", got)
	}
	ref := QAAnswerMessage{Type: MsgQaAnswer, Question: "x", Answer: "cap reached", Refused: true}
	r2, _ := json.Marshal(ref)
	var got2 QAAnswerMessage
	if err := json.Unmarshal(r2, &got2); err != nil {
		t.Fatal(err)
	}
	if !got2.Refused {
		t.Fatal("refused flag not preserved")
	}
}

func TestParseClientMessageRejectsServerTypes(t *testing.T) {
	if _, err := ParseClientMessage([]byte(`{"type":"question","content":"x","idx":1}`)); err == nil {
		t.Fatal("client sent server-type message")
	}
}
