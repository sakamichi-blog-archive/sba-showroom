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
		_, _ = fmt.Fprint(w, body)
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
	t.Cleanup(func() { os.Stdout = origStdout })

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

	_ = pw.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(pr); err != nil {
		t.Fatal(err)
	}
	_ = pr.Close()

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

func TestWatchPollInterval_NearFuture(t *testing.T) {
	// When the schedule is < 20 s away, the interval must be less than 20 s
	// so the watcher wakes near the scheduled time rather than oversleeping.
	ts := time.Now().Add(8 * time.Second).Unix()
	got := watchPollInterval(ts)
	if got >= 20*time.Second {
		t.Errorf("expected interval < 20s for near-future schedule, got %v", got)
	}
}

func TestWatchRoom_SchedulePassthroughChecksStreamURLs(t *testing.T) {
	// When NextLiveSchedule has passed and is_live=false, watchRoom must poll
	// stream URLs directly (Phase 1→2) rather than waiting for is_live=true.
	pastTS := time.Now().Add(-1 * time.Minute).Unix()
	roomBody := fmt.Sprintf(`{"id":1,"url_key":"testroom","is_live":false,"next_live_schedule":%d}`, pastTS)

	streamChecked := make(chan struct{}, 1)

	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, roomBody)
	}))
	defer cdnSrv.Close()

	showroomSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case streamChecked <- struct{}{}:
		default:
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"streaming_url_list":[]}`)
	}))
	defer showroomSrv.Close()

	oldCDN := cdnBaseURL
	oldShowroom := showroomBaseURL
	cdnBaseURL = cdnSrv.URL
	showroomBaseURL = showroomSrv.URL
	defer func() {
		cdnBaseURL = oldCDN
		showroomBaseURL = oldShowroom
	}()

	ctx, cancel := context.WithCancel(context.Background())
	wt := &watcher{
		ctx:    ctx,
		cancel: cancel,
		active: make(map[string]*runner.FFmpegProcess),
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wt.watchRoom("testroom")
	}()

	select {
	case <-streamChecked:
	case <-time.After(2 * time.Second):
		t.Error("stream URL endpoint not called; expected Phase 1→2 passthrough after scheduled time")
	}
	cancel()
	wg.Wait()
}

func TestWatchRoom_StreamURLsBeforeRoomAPIAtSchedule(t *testing.T) {
	// When the schedule passes during the poll sleep, stream URLs must be
	// checked before the next fetchRoom call (priority order: stream > room).
	nearTS := time.Now().Add(time.Second).Unix() // always 0–1 s in the future

	var mu sync.Mutex
	var callOrder []string
	streamCalled := make(chan struct{}, 1)

	roomBody := fmt.Sprintf(`{"id":1,"url_key":"testroom","is_live":false,"next_live_schedule":%d}`, nearTS)

	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callOrder = append(callOrder, "room")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, roomBody)
	}))
	defer cdnSrv.Close()

	showroomSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callOrder = append(callOrder, "stream")
		mu.Unlock()
		select {
		case streamCalled <- struct{}{}:
		default:
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"streaming_url_list":[]}`)
	}))
	defer showroomSrv.Close()

	oldCDN, oldShowroom := cdnBaseURL, showroomBaseURL
	cdnBaseURL, showroomBaseURL = cdnSrv.URL, showroomSrv.URL
	defer func() { cdnBaseURL, showroomBaseURL = oldCDN, oldShowroom }()

	ctx, cancel := context.WithCancel(context.Background())
	wt := &watcher{ctx: ctx, cancel: cancel, active: make(map[string]*runner.FFmpegProcess)}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		wt.watchRoom("testroom")
	}()

	timedOut := false
	select {
	case <-streamCalled:
	case <-time.After(5 * time.Second):
		timedOut = true
	}
	cancel()
	wg.Wait()
	if timedOut {
		t.Fatal("stream URL endpoint not called after schedule passed")
	}

	mu.Lock()
	defer mu.Unlock()

	// stream must appear before the second room call in the order slice.
	firstStream, secondRoom, roomCount := -1, -1, 0
	for i, v := range callOrder {
		if v == "room" {
			roomCount++
			if roomCount == 2 {
				secondRoom = i
			}
		}
		if v == "stream" && firstStream == -1 {
			firstStream = i
		}
	}
	if firstStream == -1 {
		t.Fatalf("stream never called; order: %v", callOrder)
	}
	if secondRoom != -1 && firstStream > secondRoom {
		t.Errorf("stream called after second fetchRoom; order: %v", callOrder)
	}
}
