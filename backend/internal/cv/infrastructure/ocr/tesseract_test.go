package ocr

import (
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// scannedPDF renders text into an image-only PDF — the "scanned" case: no
// selectable text, only pixels. Requires tesseract + poppler-utils (present
// in the app image; skip elsewhere, e.g. bare CI test containers).
func scannedPDF(t *testing.T, text string) []byte {
	t.Helper()
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract not installed")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not installed")
	}

	img := image.NewGray(image.Rect(0, 0, 640, 120))
	face := basicfont.Face7x13
	d := &font.Drawer{
		Dst:  img,
		Src:  image.White,
		Face: face,
		Dot:  fixed.P(24, 60),
	}
	d.DrawString(text)

	dir := t.TempDir()
	pngPath := filepath.Join(dir, "page.png")
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	pdfPath := filepath.Join(dir, "scanned.pdf")
	imp := pdfcpu.DefaultImportConfig()
	imp.UserDim = true
	imp.PageDim = &types.Dim{Width: 640, Height: 120}
	if err := api.ImportImagesFile([]string{pngPath}, pdfPath, imp, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Functional OCR verification (carryover item 9): a scanned PDF (image-only)
// must rasterize via pdftoppm and produce text via tesseract.
func TestOCRScannedPDF(t *testing.T) {
	pdf := scannedPDF(t, "Intivai OCR fixture: senior Go engineer")
	text, err := Extract(pdf)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	low := strings.ToLower(text)
	if !strings.Contains(low, "engineer") || !strings.Contains(low, "go") {
		t.Fatalf("OCR text missing expected words: %q", text)
	}
}

// Degenerate input must fail loudly, not hang.
func TestOCRInvalidPDF(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract not installed")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not installed")
	}
	if _, err := Extract([]byte("not a pdf")); err == nil {
		t.Fatal("invalid pdf accepted")
	}
}
