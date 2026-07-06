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

func TestSelectStream_PrefersRTMP(t *testing.T) {
	items := []streamingURLItem{
		{Type: "rtmp", Quality: 100, URL: "rtmp://example.com/live", StreamName: "stream1"},
		{Type: "hls", Quality: 100, URL: "https://example.com/live.m3u8"},
	}
	url, typ, ok := selectStream(items, false)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if typ != "rtmp" {
		t.Errorf("type: got %q, want rtmp", typ)
	}
	if url != "rtmp://example.com/live/stream1" {
		t.Errorf("url: got %q", url)
	}
}

func TestSelectStream_PrefersHLS(t *testing.T) {
	items := []streamingURLItem{
		{Type: "rtmp", Quality: 100, URL: "rtmp://example.com/live", StreamName: "stream1"},
		{Type: "hls", Quality: 100, URL: "https://example.com/live.m3u8"},
	}
	url, typ, ok := selectStream(items, true)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if typ != "hls" {
		t.Errorf("type: got %q, want hls", typ)
	}
	if url != "https://example.com/live.m3u8" {
		t.Errorf("url: got %q", url)
	}
}

func TestSelectStream_FallsBackToRTMP(t *testing.T) {
	// HLS preferred but unavailable — should fall back to RTMP.
	items := []streamingURLItem{
		{Type: "rtmp", Quality: 100, URL: "rtmp://example.com/live", StreamName: "stream1"},
	}
	_, typ, ok := selectStream(items, true)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if typ != "rtmp" {
		t.Errorf("type: got %q, want rtmp", typ)
	}
}

func TestSelectStream_FallsBackToHLS(t *testing.T) {
	// RTMP preferred but unavailable — should fall back to HLS.
	items := []streamingURLItem{
		{Type: "hls", Quality: 100, URL: "https://example.com/live.m3u8"},
	}
	_, typ, ok := selectStream(items, false)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if typ != "hls" {
		t.Errorf("type: got %q, want hls", typ)
	}
}

func TestSelectStream_PicksHighestQuality(t *testing.T) {
	items := []streamingURLItem{
		{Type: "rtmp", Quality: 10, URL: "rtmp://example.com/live", StreamName: "low"},
		{Type: "rtmp", Quality: 100, URL: "rtmp://example.com/live", StreamName: "high"},
		{Type: "rtmp", Quality: 50, URL: "rtmp://example.com/live", StreamName: "mid"},
	}
	url, _, ok := selectStream(items, false)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if url != "rtmp://example.com/live/high" {
		t.Errorf("url: got %q, want highest quality stream", url)
	}
}

func TestSelectStream_Empty(t *testing.T) {
	_, _, ok := selectStream(nil, false)
	if ok {
		t.Error("expected ok=false for empty list")
	}
}
