package tools

import (
	"os"
	"strings"
	"testing"
)

func TestIsPDF(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want bool
	}{
		{"pdf header", []byte("%PDF-1.7\n..."), true},
		{"pdf header with leading whitespace", []byte("\n %PDF-1.4"), true},
		{"plain text", []byte("hello world"), false},
		{"empty", []byte(""), false},
		{"png", []byte("\x89PNG\r\n"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isPDF(c.data); got != c.want {
				t.Errorf("isPDF(%q) = %v, want %v", c.data, got, c.want)
			}
		})
	}
}

func TestExtractPDFText(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.pdf")
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	text, err := extractPDFText(data)
	if err != nil {
		t.Fatalf("extractPDFText: %v", err)
	}
	if !strings.Contains(text, "This is a heading") || !strings.Contains(text, "This is content") {
		t.Errorf("extracted text missing expected content; got: %q", text)
	}
}

// A corrupted/non-PDF blob must fail gracefully (error, never a panic).
func TestExtractPDFText_Garbage(t *testing.T) {
	_, err := extractPDFText([]byte("%PDF-1.5\nnot really a pdf at all"))
	if err == nil {
		t.Error("expected an error for a malformed PDF, got nil")
	}
}
