package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Extract rasterizes the input PDF into 200dpi PNGs with pdftoppm,
// then runs tesseract over each page image.
func Extract(pdf []byte) (string, error) {
	return ExtractContext(context.Background(), pdf)
}

// ExtractContext rasterizes the input PDF into 200dpi PNGs with pdftoppm,
// then runs tesseract over each page image using the provided context.
func ExtractContext(ctx context.Context, pdf []byte) (string, error) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", fmt.Errorf("tesseract not installed: %w", err)
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return "", fmt.Errorf("pdftoppm not installed: %w", err)
	}

	dir, err := os.MkdirTemp("", "ocr-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	in := filepath.Join(dir, "cv.pdf")
	if err := os.WriteFile(in, pdf, 0o600); err != nil {
		return "", err
	}

	prefix := filepath.Join(dir, "page")
	if out, err := exec.CommandContext(ctx, "pdftoppm", "-png", "-r", "200", in, prefix).CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftoppm failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	pages, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		return "", err
	}
	sort.Strings(pages)
	if len(pages) == 0 {
		return "", errors.New("pdftoppm produced no pages")
	}

	var sb strings.Builder
	for _, page := range pages {
		outPath := filepath.Join(dir, "text")
		cmd := exec.CommandContext(ctx, "tesseract", page, outPath, "-l", "eng")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("tesseract failed: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		txt, err := os.ReadFile(outPath + ".txt")
		if err != nil {
			return "", fmt.Errorf("read ocr output: %w", err)
		}
		sb.Write(txt)
		sb.WriteString("\n")
	}
	return sb.String(), nil
}
