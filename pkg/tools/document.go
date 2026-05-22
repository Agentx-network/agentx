package tools

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
)

// isPDF reports whether the bytes look like a PDF (magic header "%PDF-").
// Detecting by content, not extension, so a mislabelled or extension-less file
// is still handled correctly.
func isPDF(data []byte) bool {
	return bytes.HasPrefix(bytes.TrimLeft(data, " \t\r\n"), []byte("%PDF-"))
}

// extractPDFText pulls the readable text out of a PDF's raw bytes. A PDF is a
// compressed binary container, so handing its raw bytes to the model yields
// garbage — this decompresses the content streams and returns the text in
// reading order. Scanned/image-only PDFs have no embedded text and will return
// an empty result (they'd need OCR, which this does not do).
//
// The underlying parser can panic on malformed/encrypted PDFs, so extraction is
// wrapped in a recover to fail gracefully instead of crashing the gateway.
func extractPDFText(data []byte) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("could not parse PDF (it may be encrypted, scanned, or corrupted): %v", r)
		}
	}()

	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("could not read PDF: %w", err)
	}
	plain, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("could not extract PDF text: %w", err)
	}
	var sb strings.Builder
	if _, err := io.Copy(&sb, plain); err != nil {
		return "", fmt.Errorf("could not read extracted PDF text: %w", err)
	}
	return strings.TrimSpace(sb.String()), nil
}
