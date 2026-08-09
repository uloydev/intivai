package domain

import (
	"strings"
)

var stopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "that": true, "this": true,
	"from": true, "have": true, "has": true, "are": true, "was": true, "were": true,
	"will": true, "your": true, "our": true, "their": true, "they": true, "them": true,
	"about": true, "into": true, "over": true, "under": true, "also": true, "very": true,
	"just": true, "not": true, "can": true, "all": true, "any": true, "who": true,
	"what": true, "when": true, "where": true, "which": true, "while": true, "been": true,
	"being": true, "would": true, "could": true, "should": true, "more": true,
	"most": true, "some": true, "such": true, "than": true, "then": true, "there": true,
	"these": true, "those": true, "through": true, "using": true, "used": true, "use": true,
	"company": true, "companies": true,
	"team": true, "role": true, "including": true, "within": true, "across": true, "various": true,
}

// SemanticScore — keyword overlap fallback until fastembed embeddings land (M2.5).
// Score = fraction of job terms covered by the candidate text. Deterministic.
func SemanticScore(candidateText, jobText string) float64 {
	cand := tokenize(candidateText)
	job := tokenize(jobText)
	if len(job) == 0 {
		return 1.0
	}
	matched := 0
	for term := range job {
		if cand[term] {
			matched++
		}
	}
	return float64(matched) / float64(len(job))
}

func tokenize(text string) map[string]bool {
	out := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(text)) {
		w = strings.Trim(w, ".,;:!?()[]{}\"'")
		if len(w) < 4 || stopwords[w] {
			continue
		}
		out[w] = true
	}
	return out
}
