package attach

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tiny valid PNG (1x1) so content sniffing passes for image tests.
var pngBytes = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

func TestClassify(t *testing.T) {
	cases := map[string]Kind{
		"a.png": KindImage, "b.JPG": KindImage, "c.webp": KindImage,
		"d.mp3": KindAudio, "e.wav": KindAudio,
		"f.pdf": KindDoc, "g.txt": KindDoc, "h.csv": KindDoc,
		"v.mp4": KindVideo, "w.MOV": KindVideo, "x.webm": KindVideo, "y.mkv": KindVideo,
		"i.exe": KindUnsupported, "j": KindUnsupported, "k.zip": KindUnsupported,
	}
	for name, want := range cases {
		if got := Classify(name); got != want {
			t.Errorf("Classify(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestValidateAndStore(t *testing.T) {
	ws := t.TempDir()

	// Happy path: a real PNG stores under uploads/ and is classified image.
	src := filepath.Join(t.TempDir(), "photo.png")
	if err := os.WriteFile(src, pngBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := ValidateAndStore(src, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.Kind != KindImage || st.Name != "photo.png" {
		t.Errorf("unexpected stored: %+v", st)
	}
	if !strings.HasPrefix(st.Path, filepath.Join(ws, UploadsDirName)) {
		t.Errorf("stored outside uploads dir: %s", st.Path)
	}
	if _, err := os.Stat(st.Path); err != nil {
		t.Errorf("stored file missing: %v", err)
	}

	// Video: an mp4 (ftyp box header) stores and classifies as video.
	mp4 := filepath.Join(t.TempDir(), "clip.mp4")
	// "....ftypisom" — enough for http.DetectContentType to return video/mp4.
	os.WriteFile(mp4, append([]byte{0, 0, 0, 0x18}, []byte("ftypisom\x00\x00\x00\x00isommp42")...), 0o644)
	vst, verr := ValidateAndStore(mp4, ws)
	if verr != nil {
		t.Fatalf("unexpected error storing mp4: %v", verr)
	}
	if vst.Kind != KindVideo {
		t.Errorf("expected KindVideo, got %q", vst.Kind)
	}

	// Extension spoof: .png that isn't image bytes is rejected by the sniff.
	spoof := filepath.Join(t.TempDir(), "evil.png")
	os.WriteFile(spoof, []byte("#!/bin/sh\nrm -rf /\n"), 0o644)
	if _, err := ValidateAndStore(spoof, ws); err == nil {
		t.Error("expected spoofed .png to be rejected")
	}

	// Unsupported extension.
	bad := filepath.Join(t.TempDir(), "x.exe")
	os.WriteFile(bad, []byte("MZ"), 0o644)
	if _, err := ValidateAndStore(bad, ws); err == nil {
		t.Error("expected .exe to be rejected")
	}

	// Oversized image.
	old := MaxImageBytes
	MaxImageBytes = 10
	defer func() { MaxImageBytes = old }()
	if _, err := ValidateAndStore(src, ws); err == nil {
		t.Error("expected oversized image to be rejected")
	}
}

func TestConfineToUploads(t *testing.T) {
	ws := t.TempDir()
	uploads := filepath.Join(ws, UploadsDirName)
	os.MkdirAll(uploads, 0o755)

	inside := filepath.Join(uploads, "ok.png")
	os.WriteFile(inside, pngBytes, 0o644)
	if err := ConfineToUploads(inside, ws); err != nil {
		t.Errorf("expected inside path to pass, got %v", err)
	}

	// Path traversal / outside paths must be rejected.
	for _, p := range []string{
		"/etc/passwd",
		filepath.Join(uploads, "..", "secret.txt"),
		filepath.Join(ws, "other.txt"),
	} {
		if err := ConfineToUploads(p, ws); err == nil {
			t.Errorf("expected %q to be rejected", p)
		}
	}
}

func TestDisplayBlockRoundTrip(t *testing.T) {
	media := []string{"/ws/uploads/upload-1-cat.png", "/ws/uploads/upload-2-doc.pdf"}
	block := EncodeDisplayBlock(media)
	if !strings.HasPrefix(block, DisplayMarker) {
		t.Fatalf("block missing marker: %q", block)
	}
	full := "look at these" + block
	if got := StripDisplayBlock(full); got != "look at these" {
		t.Errorf("StripDisplayBlock = %q, want %q", got, "look at these")
	}
	// No marker → unchanged.
	if got := StripDisplayBlock("plain text"); got != "plain text" {
		t.Errorf("StripDisplayBlock(plain) = %q", got)
	}
	// Empty media → empty block.
	if EncodeDisplayBlock(nil) != "" {
		t.Error("expected empty block for no media")
	}
}

func TestMimeForPath(t *testing.T) {
	cases := map[string]string{
		"a.png": "image/png", "b.jpg": "image/jpeg", "c.gif": "image/gif",
		"d.mp3": "audio/mpeg", "e.wav": "audio/wav",
	}
	for name, want := range cases {
		if got := MimeForPath(name, nil); got != want {
			t.Errorf("MimeForPath(%q) = %q, want %q", name, got, want)
		}
	}
}
