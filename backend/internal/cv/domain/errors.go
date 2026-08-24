package domain

import "errors"

var (
	ErrNotFound = errors.New("candidate not found")

	// ErrUnreadable — the stored file could not be parsed as PDF/DOCX nor
	// OCR'd (true OCR exhaustion). Terminal failed_ocr outcome; wrapping
	// causes keeps their detail for logs while SafeFailureMessage maps them
	// to candidate-safe categories (finding D26).
	ErrUnreadable = errors.New("cv unreadable")
)
