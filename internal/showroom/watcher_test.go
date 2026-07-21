package showroom

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sakamichi-blog-archive/sba-showroom/internal/runner"
)

func TestWatchRoom_LogsScheduledImmediately(t *testing.T) {
	// After the initial fetchRoom, if NextLiveSchedule is non-zero the watcher
	// must log "Scheduled:" before sleeping for the first poll interval (20 s).
	// Cancel after 300 ms — well before any sleep fires — and verify the log
	// appeared in output.
	futureTS := time.Now().Add(2 * time.Hour).Unix()
	body := fmt.Sprintf(`{"id":1,"url_key":"testroom","next_live_schedule":%d}`, futureTS)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	old := cdnBaseURL
	cdnBaseURL = srv.URL
	defer func() { cdnBaseURL = old }()

	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	origStdout := os.Stdout
	os.Stdout = pw

	ctx, cancel := context.WithCancel(context.Background())
	w := &watcher{
		ctx:    ctx,
		cancel: cancel,
		active: make(map[string]*runner.FFmpegProcess),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		w.watchRoom("testroom")
	}()

	time.Sleep(300 * time.Millisecond)
	cancel()
	wg.Wait()

	pw.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(pr); err != nil {
		t.Fatal(err)
	}
	pr.Close()

	if !strings.Contains(buf.String(), "Scheduled:") {
		t.Errorf("expected 'Scheduled:' in output before first poll interval; got: %q", buf.String())
	}
}

func TestWatchPollInterval(t *testing.T) {
	tests := []struct {
		name   string
		offset time.Duration // positive = future, negative = past
		want   time.Duration
	}{
		{"no schedule", 0, 20 * time.Second},
		{"future", 10 * time.Minute, 20 * time.Second},
		{"2 min overdue", -2 * time.Minute, 4 * time.Second},
		{"6 min overdue", -6 * time.Minute, 8 * time.Second},
		{"25 min overdue", -25 * time.Minute, 20 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ts int64
			if tt.offset != 0 {
				ts = time.Now().Add(tt.offset).Unix()
			}
			got := watchPollInterval(ts)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
