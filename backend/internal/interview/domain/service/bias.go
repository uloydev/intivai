package service

import "strings"

// Protected-class bias rules. Questions must never probe these — the
// system prompt bans them AND generated questions are filtered here.
var biasRules = []struct {
	Category string
	Keywords []string
}{
	{"age", []string{"how old", "age", "birth year", "born in", "years old", "generation z", "millennial"}},
	{"marital_family", []string{"marital", "married", "spouse", "husband", "wife", "children", "pregnant", "pregnancy", "family plans", "caretaker"}},
	{"religion", []string{"religion", "religious", "church", "mosque", "temple", "faith", "pray"}},
	{"political", []string{"political", "politics", "political party", "vote", "voting", "government"}},
	{"gender", []string{"gender", "sex", "gender identity", "man or woman", "male or female", "motherhood", "a man", "a woman"}},
	{"ethnicity", []string{"race", "ethnic", "ethnicity", "skin color"}},
	{"disability", []string{"disability", "disabled", "medical condition", "mental health", "handicap"}},
}

// DetectBias returns the protected categories a text probes. Empty slice =
// clean.
func DetectBias(text string) []string {
	lower := strings.ToLower(text)
	flagged := []string{}
	for _, rule := range biasRules {
		for _, kw := range rule.Keywords {
			if strings.Contains(lower, kw) {
				flagged = append(flagged, rule.Category)
				break
			}
		}
	}
	return flagged
}

// IsBiased is a convenience predicate for filters.
func IsBiased(text string) bool {
	return len(DetectBias(text)) > 0
}
