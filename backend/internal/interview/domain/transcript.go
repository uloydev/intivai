package domain

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
func (iv *Interview) TranscriptPairs() []TranscriptPair {
	pairs := make([]TranscriptPair, 0, len(iv.Answers))
	for _, a := range iv.Answers {
		if a.Idx < 1 || a.Idx > len(iv.Questions) {
			continue
		}
		q := iv.Questions[a.Idx-1]
		pairs = append(pairs, TranscriptPair{
			Idx:      a.Idx,
			Category: q.Category,
			Question: q.Content,
			Answer:   a.Content,
		})
	}
	return pairs
}
