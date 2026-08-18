package showroom

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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

func TestBuildFileName_UsesJST(t *testing.T) {
	// 2024-03-15 00:00 UTC = 2024-03-15 09:00 JST — date must be 240315 in JST.
	ts := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC).Unix()
	name := buildFileName("room", ts)
	if !strings.HasPrefix(name, "240315-") {
		t.Errorf("expected JST date prefix 240315, got %q", name)
	}

	// 2024-03-14 15:00 UTC = 2024-03-15 00:00 JST — still 240315 in JST.
	ts2 := time.Date(2024, 3, 14, 15, 0, 0, 0, time.UTC).Unix()
	name2 := buildFileName("room", ts2)
	if !strings.HasPrefix(name2, "240315-") {
		t.Errorf("expected JST date prefix 240315 for UTC previous day, got %q", name2)
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

func TestWaitForLive_SchedulePassthrough(t *testing.T) {
	// When the scheduled time has already passed, waitForLive must return
	// immediately without calling fetchRoom — matching sba-stream Phase 1→2.
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		http.Error(w, "fetchRoom must not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := cdnBaseURL
	cdnBaseURL = srv.URL
	defer func() { cdnBaseURL = old }()

	pastTS := time.Now().Add(-5 * time.Minute).Unix()
	room := &roomAPI{ID: 1, URLKey: "testroom"}
	if err := waitForLive(context.Background(), room, "testroom", &pastTS); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called.Load() {
		t.Error("fetchRoom was called; expected immediate return when schedule has passed")
	}
}

func TestWaitForLive_SchedulePassthrough_Immediate(t *testing.T) {
	// Verify the passthrough returns fast (not blocking on a 20s sleep).
	old := cdnBaseURL
	cdnBaseURL = "http://127.0.0.1:0" // unreachable; test must not reach HTTP
	defer func() { cdnBaseURL = old }()

	pastTS := time.Now().Add(-1 * time.Second).Unix()
	room := &roomAPI{ID: 1, URLKey: "testroom"}

	start := time.Now()
	if err := waitForLive(context.Background(), room, "testroom", &pastTS); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("passthrough took %v; expected near-instant return", elapsed)
	}
}

func TestWaitForLive_ContextCancelledDuringSleep(t *testing.T) {
	// With no schedule, waitForLive sleeps 20 s before polling. A context that
	// times out in 50 ms should unblock the sleep and return an error quickly.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var ts int64
	room := &roomAPI{ID: 1, URLKey: "testroom"}

	start := time.Now()
	err := waitForLive(ctx, room, "testroom", &ts)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("expected context error, got nil")
	}
	if elapsed > time.Second {
		t.Errorf("context cancellation took %v; expected under 1 s", elapsed)
	}
}

func TestWaitForLive_PassthroughPropagatesCancelledContext(t *testing.T) {
	// When the scheduled time has passed AND the context is already cancelled,
	// waitForLive must return the context error rather than nil so callers do
	// not proceed into runDownloadLoop after shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling

	pastTS := time.Now().Add(-5 * time.Minute).Unix()
	room := &roomAPI{ID: 1, URLKey: "testroom"}
	err := waitForLive(ctx, room, "testroom", &pastTS)
	if err == nil {
		t.Error("expected context error when ctx is cancelled on passthrough, got nil")
	}
}

func TestWaitForLive_RoomScheduleTriggersPassthrough(t *testing.T) {
	// If room.NextLiveSchedule is already in the past when waitForLive is
	// called, expectedTS is set from the room field and the passthrough fires
	// on the first loop iteration without making any HTTP calls.
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		http.Error(w, "fetchRoom must not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()

	old := cdnBaseURL
	cdnBaseURL = srv.URL
	defer func() { cdnBaseURL = old }()

	pastTS := time.Now().Add(-2 * time.Minute).Unix()
	room := &roomAPI{ID: 1, URLKey: "testroom", NextLiveSchedule: pastTS}
	var ts int64 // zero; will be overwritten by room.NextLiveSchedule
	if err := waitForLive(context.Background(), room, "testroom", &ts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called.Load() {
		t.Error("fetchRoom was called; expected immediate passthrough")
	}
	if ts != pastTS {
		t.Errorf("expectedTS not updated from NextLiveSchedule: got %d, want %d", ts, pastTS)
	}
}

func TestParseRoomURLKey(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"https://www.showroom-live.com/46_shibatayuna", "46_shibatayuna", false},
		{"https://www.showroom-live.com/r/46_shibatayuna", "46_shibatayuna", false},
		{"https://www.showroom-live.com/46_shibatayuna?some=query", "46_shibatayuna", false},
		{"https://www.showroom-live.com/r/46_shibatayuna?foo=bar", "46_shibatayuna", false},
		{"https://www.showroom-live.com/46_shibatayuna#section", "46_shibatayuna", false},
		{"https://www.showroom-live.com/46_shibatayuna/", "46_shibatayuna", false},
		{"https://www.showroom-live.com/r/46_shibatayuna/", "46_shibatayuna", false},
		{"46_shibatayuna", "46_shibatayuna", false},
		{"MY-room_1", "MY-room_1", false},
		{"https://www.showroom-live.com/", "", true},
		{"46_shibatayuna/extra", "", true},
		{"46 shibatayuna", "", true},
		{"", "", true},
		{"https://example.com/46_shibatayuna", "", true},
		{"not a url", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseRoomURLKey(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRoomURLKey(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseRoomURLKey(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
