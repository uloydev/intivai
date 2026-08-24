package domain

import (
	"fmt"
	"strings"
)

// TranscriptPair — one question/answer for the evaluator. Lives in the
// interview domain (it IS interview data); the evaluator consumes it.
type TranscriptPair struct {
	Idx      int    `json:"idx"`
	Category string `json:"category"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// TranscriptPairs builds ordered question/answer pairs from the transcript.
// Answers outside the question range are skipped (defensive).
// Multi-turn exchanges for a question topic are combined with structured turn markers.
func (iv *Interview) TranscriptPairs() []TranscriptPair {
	ansByQ := make(map[int][]Answer)
	for _, a := range iv.Answers {
		if a.Idx >= 1 && a.Idx <= len(iv.Questions) {
			ansByQ[a.Idx] = append(ansByQ[a.Idx], a)
		}
	}

	pairs := make([]TranscriptPair, 0, len(iv.Questions))
	for _, q := range iv.Questions {
		ansList := ansByQ[q.Idx]
		if len(ansList) == 0 {
			continue
		}
		if len(ansList) == 1 && ansList[0].Turn <= 1 && (ansList[0].Action == "" || ansList[0].Action == "advance") {
			pairs = append(pairs, TranscriptPair{
				Idx:      q.Idx,
				Category: q.Category,
				Question: q.Content,
				Answer:   ansList[0].Content,
			})
			continue
		}

		var sb strings.Builder
		for i, a := range ansList {
			if i > 0 {
				sb.WriteString("\n---\n")
			}
			turnNum := a.Turn
			if turnNum == 0 {
				turnNum = i + 1
			}
			act := a.Action
			if act == "" {
				act = "reply"
			}
			fmt.Fprintf(&sb, "[Candidate Turn %d (%s)]: %s", turnNum, act, a.Content)
		}
		pairs = append(pairs, TranscriptPair{
			Idx:      q.Idx,
			Category: q.Category,
			Question: q.Content,
			Answer:   sb.String(),
		})
	}
	return pairs
}
