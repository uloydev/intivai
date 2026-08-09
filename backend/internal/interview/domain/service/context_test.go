package service

import "testing"

func TestTrimContextKeepsLastTenPairs(t *testing.T) {
	msgs := make([]ContextMessage, 0, 22)
	msgs = append(msgs, ContextMessage{Role: RoleSystem, Content: "sys"})
	for i := 1; i <= 11; i++ {
		msgs = append(msgs,
			ContextMessage{Role: RoleAssistant, Content: q(i)},
			ContextMessage{Role: RoleUser, Content: a(i)},
		)
	}
	got := TrimContext(msgs, DefaultContextWindow)
	want := 1 + 2*DefaultContextWindow
	if len(got) != want {
		t.Fatalf("len = %d, want %d (system + last 10 pairs)", len(got), want)
	}
	if got[0].Role != RoleSystem || got[0].Content != "sys" {
		t.Fatalf("leading system lost: %+v", got[0])
	}
	if got[1].Role != RoleAssistant || got[1].Content != q(2) {
		t.Fatalf("first kept message = %+v, want question 2", got[1])
	}
	if got[len(got)-1].Role != RoleUser || got[len(got)-1].Content != a(11) {
		t.Fatalf("last message = %+v, want answer 11", got[len(got)-1])
	}
}

func TestTrimContextUnderWindowKeepsAll(t *testing.T) {
	msgs := []ContextMessage{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleAssistant, Content: "q1"},
		{Role: RoleUser, Content: "a1"},
	}
	got := TrimContext(msgs, DefaultContextWindow)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
}

func TestTrimContextWindowZeroKeepsSystemOnly(t *testing.T) {
	msgs := []ContextMessage{
		{Role: RoleSystem, Content: "sys"},
		{Role: RoleAssistant, Content: "q1"},
		{Role: RoleUser, Content: "a1"},
	}
	got := TrimContext(msgs, 0)
	if len(got) != 1 || got[0].Content != "sys" {
		t.Fatalf("got %+v, want system only", got)
	}
}

func TestTrimContextEmpty(t *testing.T) {
	if got := TrimContext(nil, DefaultContextWindow); got != nil {
		t.Fatalf("want nil, got %+v", got)
	}
}

func TestCountContextTokens(t *testing.T) {
	msgs := []ContextMessage{
		{Role: RoleSystem, Content: "12345"},   // 5
		{Role: RoleAssistant, Content: "123"},  // 3
		{Role: RoleUser, Content: "123456789"}, // 9
	}
	got := CountContextTokens(func(s string) int { return len(s) }, msgs)
	if got != 17 {
		t.Fatalf("tokens = %d, want 17 (5+3+9)", got)
	}
}

func TestCountContextTokensEmpty(t *testing.T) {
	if got := CountContextTokens(func(s string) int { return 1 }, nil); got != 0 {
		t.Fatalf("tokens = %d, want 0", got)
	}
}

func TestBudgetExceeded(t *testing.T) {
	msgs := []ContextMessage{{Role: RoleUser, Content: "xxxxxxxxxx"}}
	if !ExceedsBudget(msgs, 5, func(s string) int { return len(s) }) {
		t.Fatal("10 tokens vs budget 5 must exceed")
	}
	if ExceedsBudget(msgs, 20, func(s string) int { return len(s) }) {
		t.Fatal("10 tokens vs budget 20 must fit")
	}
}

func q(i int) string { return "question " + string(rune('0'+i)) }
func a(i int) string { return "answer " + string(rune('0'+i)) }
