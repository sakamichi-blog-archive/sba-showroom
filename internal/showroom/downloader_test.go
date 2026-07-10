package showroom

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"46_iwamotorenka", "46_iwamotorenka"},
		{"hello world", "hello_world"},
		{"Test-Room", "Test-Room"},
		{"日本語タイトル", "_______"},
		{"room/name", "room_name"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sanitizeName(tt.in)
		if got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBuildFileName(t *testing.T) {
	ts := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC).Unix()
	name := buildFileName("46_iwamotorenka", ts)

	if !strings.HasPrefix(name, "240315-46_iwamotorenka-") {
		t.Errorf("unexpected prefix in %q", name)
	}
	suffix := strings.TrimPrefix(name, "240315-46_iwamotorenka-")
	if len(suffix) != 4 {
		t.Errorf("expected 4-char hex suffix, got %q", suffix)
	}
	for _, r := range suffix {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Errorf("suffix %q contains non-hex char %q", suffix, r)
		}
	}
}

func TestBuildFileName_Uniqueness(t *testing.T) {
	ts := time.Now().Unix()
	a := buildFileName("room", ts)
	b := buildFileName("room", ts)
	if a == b {
		t.Errorf("expected unique file names, got identical: %q", a)
	}
}

func TestNoRetryOutcome(t *testing.T) {
	dir := t.TempDir()
	ffmpegErr := fmt.Errorf("ffmpeg exited with code 1")

	emptyFile := filepath.Join(dir, "empty.mp4")
	if err := os.WriteFile(emptyFile, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	fullFile := filepath.Join(dir, "full.mp4")
	if err := os.WriteFile(fullFile, []byte{0x00}, 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		runErr  error
		outPath string
		wantErr bool
	}{
		{"error/has_content", ffmpegErr, fullFile, false},
		{"error/missing", ffmpegErr, filepath.Join(dir, "missing.mp4"), true},
		{"error/empty", ffmpegErr, emptyFile, true},
		{"success/has_content", nil, fullFile, false},
		{"success/missing", nil, filepath.Join(dir, "missing.mp4"), true},
		{"success/empty", nil, emptyFile, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := noRetryOutcome(tt.runErr, tt.outPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("noRetryOutcome() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileHasContent(t *testing.T) {
	dir := t.TempDir()

	// missing file
	if fileHasContent(filepath.Join(dir, "missing.mp4")) {
		t.Error("expected false for missing file")
	}

	// directory at path
	if fileHasContent(dir) {
		t.Error("expected false for directory")
	}

	// empty file
	empty := filepath.Join(dir, "empty.mp4")
	if err := os.WriteFile(empty, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	if fileHasContent(empty) {
		t.Error("expected false for empty file")
	}

	// file with content
	full := filepath.Join(dir, "full.mp4")
	if err := os.WriteFile(full, []byte{0x00}, 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileHasContent(full) {
		t.Error("expected true for file with content")
	}
}

func TestResolveHLSURL_PicksHighestQuality(t *testing.T) {
	items := []streamingURLItem{
		{Type: "hls", Quality: 10, URL: "https://example.com/low.m3u8"},
		{Type: "hls", Quality: 100, URL: "https://example.com/high.m3u8"},
		{Type: "hls", Quality: 50, URL: "https://example.com/mid.m3u8"},
	}

	var best *streamingURLItem
	for i := range items {
		item := &items[i]
		if item.Type == "hls" && (best == nil || item.Quality > best.Quality) {
			best = item
		}
	}

	if best == nil || best.URL != "https://example.com/high.m3u8" {
		t.Errorf("expected highest quality HLS URL, got %v", best)
	}
}

func TestResolveHLSURL_NoHLS(t *testing.T) {
	items := []streamingURLItem{}

	var best *streamingURLItem
	for i := range items {
		item := &items[i]
		if item.Type == "hls" && (best == nil || item.Quality > best.Quality) {
			best = item
		}
	}

	if best != nil {
		t.Errorf("expected nil for empty list, got %v", best)
	}
}
