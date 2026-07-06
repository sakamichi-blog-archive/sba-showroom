package showroom

import (
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
