package domain

import (
	"encoding/json"
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
}

func TestParseClientMessageRejectsServerTypes(t *testing.T) {
	if _, err := ParseClientMessage([]byte(`{"type":"question","content":"x","idx":1}`)); err == nil {
		t.Fatal("client sent server-type message")
	}
}
