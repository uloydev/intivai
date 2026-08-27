package domain

import "errors"

var (
	ErrNotFound         = errors.New("interview not found")
	ErrNoClock          = errors.New("interview has no clock")
	ErrEvaluationExists = errors.New("evaluation already exists")
	ErrQAActive         = errors.New("interview is not active for QA append")
)
