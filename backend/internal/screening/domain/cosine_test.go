package domain

import (
	"math"
	"testing"
)

func TestCosineIdentical(t *testing.T) {
	v := []float32{1, 0, 0}
	if s := Cosine(v, v); math.Abs(s-1) > 1e-9 {
		t.Fatalf("cosine(identical) = %f, want 1", s)
	}
}

func TestCosineOrthogonal(t *testing.T) {
	if s := Cosine([]float32{1, 0}, []float32{0, 1}); math.Abs(s) > 1e-9 {
		t.Fatalf("cosine(orthogonal) = %f, want 0", s)
	}
}

func TestCosinePartial(t *testing.T) {
	a := []float32{1, 1}
	b := []float32{1, 0}
	want := 1 / math.Sqrt2
	if s := Cosine(a, b); math.Abs(s-want) > 1e-9 {
		t.Fatalf("cosine = %f, want %f", s, want)
	}
}

func TestCosineEmptyOrMismatched(t *testing.T) {
	if Cosine(nil, []float32{1}) != 0 {
		t.Fatal("empty must score 0")
	}
	if Cosine([]float32{1, 2}, []float32{1}) != 0 {
		t.Fatal("mismatched dims must score 0")
	}
}

func TestSemanticScoreWithEmbedderClamped(t *testing.T) {
	if s := SemanticScoreWithEmbedder([]float32{1}, []float32{-1}); s != 0 {
		t.Fatalf("negative cosine must clamp to 0, got %f", s)
	}
	if s := SemanticScoreWithEmbedder([]float32{1}, []float32{1}); s != 1 {
		t.Fatalf("identical vectors must score 1, got %f", s)
	}
}
