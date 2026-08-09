package application

import (
	"fmt"
	"strings"
	"testing"
)

func TestExtractPDFTextValid(t *testing.T) {
	pdf := buildMinimalPDF("Senior Go engineer with PostgreSQL experience")
	text, err := extractPDFText(pdf)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if !strings.Contains(text, "PostgreSQL") {
		t.Fatalf("text missing content: %q", text)
	}
}

func TestExtractPDFTextInvalid(t *testing.T) {
	if _, err := extractPDFText([]byte("not a pdf")); err == nil {
		t.Fatal("invalid pdf accepted")
	}
}

func buildMinimalPDF(text string) []byte {
	stream := fmt.Sprintf("BT /F1 12 Tf 72 720 Td (%s) Tj ET", text)
	var out strings.Builder
	out.WriteString("%PDF-1.4\n")
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return []byte(out.String())
}
