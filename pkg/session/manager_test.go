package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendTranscript(t *testing.T) {
	dir := t.TempDir()
	sm := NewSessionManager(dir)

	// Append a few messages; the transcript is append-only and independent of
	// the summarized session .json.
	sm.AppendTranscript("agent:main:main", "user", "hello")
	sm.AppendTranscript("agent:main:main", "assistant", "hi there")
	sm.AppendTranscript("agent:main:main", "user", "what is 2+2?")
	sm.AppendTranscript("agent:main:main", "user", "") // empty content — skipped

	path := filepath.Join(dir, "agent_main_main.transcript.jsonl")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("transcript not written: %v", err)
	}
	defer f.Close()

	var got []TranscriptEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e TranscriptEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("bad jsonl line %q: %v", line, err)
		}
		got = append(got, e)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 transcript entries (empty skipped), got %d", len(got))
	}
	if got[0].Role != "user" || got[0].Content != "hello" {
		t.Errorf("entry 0 = %+v", got[0])
	}
	if got[1].Role != "assistant" || got[1].Content != "hi there" {
		t.Errorf("entry 1 = %+v", got[1])
	}
	if got[2].Timestamp == 0 {
		t.Errorf("expected a timestamp, got 0")
	}

	// Truncating the session history must NOT affect the transcript.
	sm.AddMessage("agent:main:main", "user", "x")
	sm.TruncateHistory("agent:main:main", 0)
	if data, _ := os.ReadFile(path); strings.Count(string(data), "\n") != 3 {
		t.Errorf("transcript should be untouched by TruncateHistory")
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"telegram:123456", "telegram_123456"},
		{"discord:987654321", "discord_987654321"},
		{"slack:C01234", "slack_C01234"},
		{"no-colons-here", "no-colons-here"},
		{"multiple:colons:here", "multiple_colons_here"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSave_WithColonInKey(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	// Create a session with a key containing colon (typical channel session key).
	key := "telegram:123456"
	sm.GetOrCreate(key)
	sm.AddMessage(key, "user", "hello")

	// Save should succeed even though the key contains ':'
	if err := sm.Save(key); err != nil {
		t.Fatalf("Save(%q) failed: %v", key, err)
	}

	// The file on disk should use sanitized name.
	expectedFile := filepath.Join(tmpDir, "telegram_123456.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("expected session file %s to exist", expectedFile)
	}

	// Load into a fresh manager and verify the session round-trips.
	sm2 := NewSessionManager(tmpDir)
	history := sm2.GetHistory(key)
	if len(history) != 1 {
		t.Fatalf("expected 1 message after reload, got %d", len(history))
	}
	if history[0].Content != "hello" {
		t.Errorf("expected message content %q, got %q", "hello", history[0].Content)
	}
}

func TestSave_RejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	sm := NewSessionManager(tmpDir)

	badKeys := []string{"", ".", "..", "foo/bar", "foo\\bar"}
	for _, key := range badKeys {
		sm.GetOrCreate(key)
		if err := sm.Save(key); err == nil {
			t.Errorf("Save(%q) should have failed but didn't", key)
		}
	}
}
