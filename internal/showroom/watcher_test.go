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
	"sync/atomic"
	"testing"
	"time"

	"github.com/sakamichi-blog-archive/sba-showroom/internal/runner"
)

func TestWatchRoom_LogsScheduledImmediately(t *testing.T) {
	// After the initial fetchRoom, if NextLiveSchedule is non-zero the watcher
	// must log "Scheduled:" before sleeping for the first poll interval (20 s).
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

	// Read from the pipe until "Scheduled:" appears rather than sleeping a
	// fixed duration, so the test is not sensitive to goroutine scheduling.
	logReceived := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		tmp := make([]byte, 256)
		for {
			n, err := pr.Read(tmp)
			buf.Write(tmp[:n])
			if strings.Contains(buf.String(), "Scheduled:") {
				logReceived <- buf.String()
				return
			}
			if err != nil {
				return
			}
		}
	}()

	timedOut := false
	var output string
	select {
	case output = <-logReceived:
	case <-time.After(2 * time.Second):
		timedOut = true
	}
	cancel()
	wg.Wait()
	_ = pw.Close()
	os.Stdout = origStdout
	_ = pr.Close()

	if timedOut {
		t.Fatal("'Scheduled:' not logged within 2 s of watchRoom start")
	}
	if !strings.Contains(output, "Scheduled:") {
		t.Errorf("expected 'Scheduled:' in output; got: %q", output)
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

func TestWatchRoom_NoDoubleDownloadAfterPassthrough(t *testing.T) {
	// After a Phase 1→2 passthrough fires (schedule passed, is_live=false) and
	// runDownload finds a stream URL, the watcher must NOT start a second
	// download when the SHOWROOM API subsequently reports is_live=true for the
	// same session (~1 min lag). The stream URL endpoint should be called exactly
	// once (for the passthrough's resolveHLS call).

	// Speed up the idle poll so the test completes in milliseconds.
	orig := defaultPollInterval
	defaultPollInterval = 100 * time.Millisecond
	t.Cleanup(func() { defaultPollInterval = orig })

	pastTS := time.Now().Add(-1 * time.Minute).Unix()

	var roomCalls atomic.Int32
	isLiveServed := make(chan struct{}, 1)

	cdnSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if roomCalls.Add(1) == 1 {
			_, _ = fmt.Fprintf(w, `{"id":1,"url_key":"testroom","is_live":false,"next_live_schedule":%d}`, pastTS)
		} else {
			select {
			case isLiveServed <- struct{}{}:
			default:
			}
			_, _ = fmt.Fprint(w, `{"id":1,"url_key":"testroom","is_live":true}`)
		}
	}))
	defer cdnSrv.Close()

	// Return a valid-looking HLS URL on the first call so resolveHLS succeeds
	// (triggering the passthrough). Count all calls; a second call means the
	// bug fired a second runDownload.
	// streamCalls counts only /api/live/streaming_url hits (resolveHLS calls),
	// not ffmpeg's subsequent m3u8 fetch against the same test server.
	var streamCalls atomic.Int32
	var showroomSrv *httptest.Server
	showroomSrv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/api/live/streaming_url" {
			http.NotFound(w, r)
			return
		}
		if streamCalls.Add(1) == 1 {
			// Use the showroom server itself as a dummy HLS URL so ffmpeg
			// (if present) fails immediately rather than timing out.
			_, _ = fmt.Fprintf(w, `{"streaming_url_list":[{"type":"hls","quality":1,"url":"%s/dummy.m3u8"}]}`, showroomSrv.URL)
		} else {
			_, _ = fmt.Fprint(w, `{"streaming_url_list":[]}`)
		}
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

	// Wait until fetchRoom has returned is_live=true at least once, meaning
	// the passthrough download completed and the watcher processed the API lag.
	select {
	case <-isLiveServed:
	case <-time.After(10 * time.Second):
		cancel()
		wg.Wait()
		t.Fatal("is_live=true never served; passthrough may not have fired")
	}

	// Allow one more poll cycle for the is_live=true response to be processed
	// before we shut down and check the count.
	time.Sleep(500 * time.Millisecond)
	cancel()
	wg.Wait()

	got := streamCalls.Load()
	if got != 1 {
		t.Errorf("stream URL endpoint called %d time(s); want 1 — duplicate download after passthrough?", got)
	}
}
