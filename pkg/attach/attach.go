// Package attach handles chat file attachments: classifying, validating, and
// storing files a user uploads so they can be threaded through the existing
// bus.InboundMessage.Media []string (paths) contract.
//
// Storage model: the desktop app copies picked files into <workspace>/uploads/
// (inside the workspace, so the read_file sandbox already permits reads) and
// passes their paths to the gateway. The gateway re-confines those paths to the
// uploads dir before the runner ever reads bytes — defense in depth against a
// buggy/compromised client passing arbitrary paths.
package attach

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Agentx-network/agentx/pkg/utils"
)

// Kind classifies an attachment by how it reaches the model.
type Kind string

const (
	KindImage       Kind = "image"       // sent to the model as vision input (FilePart)
	KindAudio       Kind = "audio"       // sent to the model / transcribed
	KindDoc         Kind = "doc"         // read via the read_file tool
	KindVideo       Kind = "video"       // processed by tools (ffmpeg/skills), not sent to the model
	KindUnsupported Kind = "unsupported" // rejected
)

// Limits. Kept as exported vars (not consts) so tests can tighten them.
var (
	// MaxFiles is the maximum number of attachments per message.
	MaxFiles = 8
	// MaxImageBytes / MaxAudioBytes / MaxDocBytes / MaxVideoBytes are per-file
	// size caps. Video is large because the use case is processing (e.g.
	// compression), not sending bytes to the model.
	MaxImageBytes int64 = 20 << 20  // 20 MB
	MaxAudioBytes int64 = 25 << 20  // 25 MB
	MaxDocBytes   int64 = 30 << 20  // 30 MB
	MaxVideoBytes int64 = 200 << 20 // 200 MB
	// MaxTotalBytes caps the combined size of all attachments on one message.
	MaxTotalBytes int64 = 300 << 20 // 300 MB (a single video can be large)
)

// UploadsDirName is the workspace subdirectory attachments are stored in.
const UploadsDirName = "uploads"

// imageExts / audioExts / docExts are the allowed extensions per kind.
// docExts mirrors the read_file-readable set in pkg/agent (docReadableExts);
// audioExts mirrors utils.IsAudioFile's list. Kept here as the single source of
// truth for what the uploader accepts.
var (
	imageExts = map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true,
	}
	audioExts = map[string]bool{
		".mp3": true, ".wav": true, ".ogg": true, ".m4a": true, ".flac": true, ".aac": true, ".wma": true,
	}
	docExts = map[string]bool{
		".pdf": true, ".txt": true, ".md": true, ".markdown": true, ".csv": true,
		".json": true, ".log": true, ".yaml": true, ".yml": true, ".xml": true,
		".html": true, ".htm": true, ".tsv": true, ".ini": true, ".toml": true,
	}
	videoExts = map[string]bool{
		".mp4": true, ".mov": true, ".webm": true, ".mkv": true, ".avi": true,
		".m4v": true, ".flv": true, ".wmv": true, ".mpeg": true, ".mpg": true,
	}
)

// mimeByExt maps known attachment extensions to their canonical MIME type,
// used when handing bytes to the model (provider APIs need an accurate type).
var mimeByExt = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
	".webp": "image/webp", ".gif": "image/gif",
	".mp3": "audio/mpeg", ".wav": "audio/wav", ".m4a": "audio/mp4",
	".ogg": "audio/ogg", ".flac": "audio/flac", ".aac": "audio/aac", ".wma": "audio/x-ms-wma",
	".mp4": "video/mp4", ".mov": "video/quicktime", ".webm": "video/webm",
	".mkv": "video/x-matroska", ".avi": "video/x-msvideo", ".m4v": "video/x-m4v",
	".flv": "video/x-flv", ".wmv": "video/x-ms-wmv", ".mpeg": "video/mpeg", ".mpg": "video/mpeg",
}

// MimeForPath returns the canonical MIME type for a path by extension, falling
// back to content sniffing, then to application/octet-stream.
func MimeForPath(path string, data []byte) string {
	if m, ok := mimeByExt[strings.ToLower(filepath.Ext(path))]; ok {
		return m
	}
	if len(data) > 0 {
		sniff := data
		if len(sniff) > 512 {
			sniff = sniff[:512]
		}
		if ct := http.DetectContentType(sniff); ct != "" {
			return ct
		}
	}
	return "application/octet-stream"
}

// Classify returns the Kind for a path based purely on its extension.
func Classify(path string) Kind {
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case imageExts[ext]:
		return KindImage
	case audioExts[ext]:
		return KindAudio
	case docExts[ext]:
		return KindDoc
	case videoExts[ext]:
		return KindVideo
	default:
		return KindUnsupported
	}
}

// maxBytesFor returns the per-file size cap for a kind.
func maxBytesFor(k Kind) int64 {
	switch k {
	case KindImage:
		return MaxImageBytes
	case KindAudio:
		return MaxAudioBytes
	case KindDoc:
		return MaxDocBytes
	case KindVideo:
		return MaxVideoBytes
	default:
		return 0
	}
}

// DisplayMarker delimits the machine-readable attachment block appended to a
// stored user message so the chat UI can re-render attachments after a reload.
// The user's real typed text always precedes it, so the visible text is
// everything before this marker. Each following line is "<kind>|<path>|<name>".
const DisplayMarker = "\n\n[[attachments]]\n"

// EncodeDisplayBlock renders media paths into a DisplayMarker block. Returns ""
// for no media so callers can append unconditionally.
func EncodeDisplayBlock(media []string) string {
	if len(media) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(DisplayMarker)
	for i, p := range media {
		if i > 0 {
			b.WriteString("\n")
		}
		// Strip any stray '|' / newlines from the name to keep the line parseable.
		name := strings.NewReplacer("|", "_", "\n", " ", "\r", " ").Replace(filepath.Base(p))
		fmt.Fprintf(&b, "%s|%s|%s", Classify(p), p, name)
	}
	return b.String()
}

// StripDisplayBlock removes the DisplayMarker block (and everything after it)
// from content, returning just the text before it. Safe on content without a
// marker.
func StripDisplayBlock(content string) string {
	if i := strings.Index(content, DisplayMarker); i >= 0 {
		return content[:i]
	}
	return content
}

// Stored describes a validated, stored attachment.
type Stored struct {
	Path string `json:"path"` // absolute path under <workspace>/uploads
	Name string `json:"name"` // original (sanitized) filename
	Size int64  `json:"size"`
	Kind Kind   `json:"kind"`
}

// sniffMatchesKind reads the first 512 bytes and checks the detected content
// type is consistent with the declared kind — blocks a .png that's really an
// executable, etc. Docs are intentionally lenient (many text formats sniff as
// "text/plain" or "application/octet-stream"); we only hard-check media.
func sniffMatchesKind(data []byte, k Kind) bool {
	ct := http.DetectContentType(data)
	switch k {
	case KindImage:
		return strings.HasPrefix(ct, "image/")
	case KindAudio:
		return strings.HasPrefix(ct, "audio/") ||
			strings.HasPrefix(ct, "video/") || // some m4a/ogg sniff as video/*
			strings.HasPrefix(ct, "application/ogg") ||
			ct == "application/octet-stream" // many codecs aren't sniffable
	case KindVideo:
		return strings.HasPrefix(ct, "video/") ||
			strings.HasPrefix(ct, "audio/") || // some containers sniff as audio/*
			ct == "application/octet-stream" // mkv/avi/etc. aren't sniffable
	case KindDoc:
		return true
	default:
		return false
	}
}

// ValidateAndStore validates a source file and, if acceptable, copies it into
// <workspace>/uploads/ under a collision-proof, sanitized name. Returns the
// stored descriptor or a user-facing error describing why it was rejected.
func ValidateAndStore(srcPath, workspace string) (*Stored, error) {
	name := utils.SanitizeFilename(srcPath)
	kind := Classify(srcPath)
	if kind == KindUnsupported {
		return nil, fmt.Errorf("%q: unsupported file type", name)
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("%q: cannot read file", name)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q: is a directory", name)
	}
	if cap := maxBytesFor(kind); info.Size() > cap {
		return nil, fmt.Errorf("%q is %s — over the %s limit for %ss",
			name, humanSize(info.Size()), humanSize(cap), kind)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return nil, fmt.Errorf("%q: cannot read file", name)
	}
	defer src.Close()

	// Sniff the first 512 bytes to guard against extension spoofing, then rewind.
	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	if !sniffMatchesKind(head[:n], kind) {
		return nil, fmt.Errorf("%q doesn't look like a valid %s file", name, kind)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("%q: cannot read file", name)
	}

	uploadsDir := filepath.Join(workspace, UploadsDirName)
	if err := os.MkdirAll(uploadsDir, 0o755); err != nil {
		return nil, fmt.Errorf("could not create uploads directory: %w", err)
	}

	// Collision-proof name: upload-<ts-nanos>-<sanitized>. Nanos so multiple
	// files picked in the same second don't clobber each other.
	dstName := fmt.Sprintf("upload-%d-%s", time.Now().UnixNano(), name)
	dstPath := filepath.Join(uploadsDir, dstName)

	// Stream-copy to a temp file then atomically rename, so a large video isn't
	// loaded fully into memory.
	tmp, err := os.CreateTemp(uploadsDir, ".upload-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("could not save %q: %w", name, err)
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, src); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("could not save %q: %w", name, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return nil, fmt.Errorf("could not save %q: %w", name, err)
	}
	tmp.Close()
	_ = os.Chmod(tmpPath, 0o644)
	if err := os.Rename(tmpPath, dstPath); err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("could not save %q: %w", name, err)
	}

	return &Stored{Path: dstPath, Name: name, Size: info.Size(), Kind: kind}, nil
}

// ConfineToUploads reports an error if path is not a regular file inside
// <workspace>/uploads/. The gateway calls this on every incoming media path
// before the runner reads bytes, so a client can never point the model at an
// arbitrary file (e.g. /etc/passwd) by lying in the /api/chat body.
func ConfineToUploads(path, workspace string) error {
	uploadsDir := filepath.Join(workspace, UploadsDirName)
	absUploads, err := filepath.Abs(uploadsDir)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absUploads, absPath)
	if err != nil || !filepath.IsLocal(rel) {
		return fmt.Errorf("attachment path is outside the uploads directory")
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("attachment not found")
	}
	if info.IsDir() {
		return fmt.Errorf("attachment path is a directory")
	}
	return nil
}

// humanSize renders a byte count as a short human string (e.g. "21.4 MB").
func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}
