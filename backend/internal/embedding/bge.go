package embedding

import (
	"context"
	"fmt"
	"sync"

	"github.com/nlpodyssey/cybertron/pkg/tasks"
	"github.com/nlpodyssey/cybertron/pkg/tasks/textencoding"
)

// Embedder produces dense vectors for semantic recall + scoring.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// BGESmall — 384-dim sentence embeddings via cybertron (pure Go, CGO-free:
// works on the alpine runtime). The model is downloaded on first use into
// ModelsDir, then inference runs fully offline. Loading is lazy — an
// unavailable model or network does not block startup; callers fall back to
// keyword recall.
//
// Default model: multi-qa-MiniLM-L6-cos-v1 (public, 384-dim, cosine — same
// architecture class as bge-small-en-v1.5, which is gated on HuggingFace).
// Set ModelName to "sentence-transformers/bge-small-en-v1.5" once HF access
// is available (or a mirror); dims stay 384.
type BGESmall struct {
	ModelsDir string
	ModelName string

	mu    sync.Mutex
	model textencoding.Interface
}

// NewBGESmall creates the embedder. Model loading happens on first Embed.
func NewBGESmall(modelsDir string) *BGESmall {
	return &BGESmall{
		ModelsDir: modelsDir,
		ModelName: "sentence-transformers/multi-qa-MiniLM-L6-cos-v1",
	}
}

// Embed encodes text with mean pooling → 384-dim vector.
func (b *BGESmall) Embed(ctx context.Context, text string) ([]float32, error) {
	m, err := b.load()
	if err != nil {
		return nil, err
	}
	result, err := m.Encode(ctx, text, 0) // 0 = mean pooling
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	vec := make([]float32, len(result.Vector.Data().F64()))
	for i, v := range result.Vector.Data().F64() {
		vec[i] = float32(v)
	}
	return vec, nil
}

func (b *BGESmall) load() (textencoding.Interface, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.model != nil {
		return b.model, nil
	}
	m, err := tasks.LoadModelForTextEncoding(&tasks.Config{
		ModelsDir: b.ModelsDir,
		ModelName: b.ModelName,
	})
	if err != nil {
		return nil, fmt.Errorf("load embedding model %s: %w", b.ModelName, err)
	}
	b.model = m
	return m, nil
}
