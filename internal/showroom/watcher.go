package showroom

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/sakamichi-blog-archive/sba-showroom/internal/runner"
)

// WatchOptions configures the Watch command.
type WatchOptions struct {
	Campaigns []string
	RoomURLs  []string // additional room URLs not covered by Campaigns
	Verbose   bool
}

type watcher struct {
	ctx     context.Context
	cancel  context.CancelFunc
	verbose bool
	mu      sync.Mutex
	active  map[string]*runner.FFmpegProcess // urlKey → running ffmpeg process
	// recording reserves a canonical url_key for the whole resolve+record span,
	// starting before resolveHLS (which active does not cover). A campaign can
	// list the same room under two key strings; Watch only dedups the raw keys,
	// so both spawn watchRoom goroutines that resolve to the same canonical
	// url_key. This reservation stops the second one from starting a concurrent
	// recording of the same stream.
	recording map[string]bool
	// watched holds the canonical url_keys already owned by a watchRoom
	// goroutine. When two raw keys resolve to the same room, the second
	// goroutine stops instead of polling redundantly for the stream's lifetime.
	watched map[string]bool
}

// claimWatch marks urlKey as owned by a watchRoom goroutine and returns true.
// It returns false if another goroutine already owns the key, signalling this
// goroutine to stop as a duplicate.
func (w *watcher) claimWatch(urlKey string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.watched == nil {
		w.watched = make(map[string]bool)
	}
	if w.watched[urlKey] {
		return false
	}
	w.watched[urlKey] = true
	return true
}

// Watch fetches rooms for each campaign slug and monitors them for live streams,
// starting concurrent downloads as rooms go live.
func Watch(opts WatchOptions) error {
	seen := make(map[string]bool)
	var roomKeys []string

	add := func(key string) {
		if !seen[key] {
			seen[key] = true
			roomKeys = append(roomKeys, key)
		}
	}

	for _, slug := range opts.Campaigns {
		keys, err := fetchCampaignRooms(context.Background(), slug)
		if err != nil {
			return fmt.Errorf("campaign %s: %w", slug, err)
		}
		for _, k := range keys {
			add(k)
		}
	}
	for _, rawURL := range opts.RoomURLs {
		key, err := parseRoomURLKey(rawURL)
		if err != nil {
			return fmt.Errorf("room URL %q: %w", rawURL, err)
		}
		add(key)
	}

	if len(roomKeys) == 0 {
		return fmt.Errorf("no rooms to watch")
	}
	if len(roomKeys) == 1 {
		fmt.Printf("Watching %s...\n", roomKeys[0])
	} else {
		fmt.Printf("Watching %s and %d other rooms...\n", roomKeys[0], len(roomKeys)-1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := &watcher{
		ctx:       ctx,
		cancel:    cancel,
		verbose:   opts.Verbose,
		active:    make(map[string]*runner.FFmpegProcess),
		recording: make(map[string]bool),
		watched:   make(map[string]bool),
	}

	for _, key := range roomKeys {
		go w.watchRoom(key)
	}

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt)

	var firstInterrupt time.Time
	for range sigCh {
		w.mu.Lock()
		n := len(w.active)
		w.mu.Unlock()

		if n == 0 {
			fmt.Println("\nNo downloads in progress. Exiting.")
			signal.Stop(sigCh)
			w.cancel()
			os.Exit(0)
		}

		now := time.Now()
		if !firstInterrupt.IsZero() && now.Sub(firstInterrupt) < 3*time.Second {
			fmt.Printf("\nStopping %d download(s)...\n", n)
			signal.Stop(sigCh)
			w.cancel()
			w.stopAll()
			w.waitForDownloads()
			os.Exit(0)
		}

		firstInterrupt = now
		fmt.Printf("\n%d download(s) in progress. Press Ctrl+C again to stop.\n", n)
	}

	return nil
}

func (w *watcher) watchRoom(urlKey string) {
	logf := func(format string, args ...any) {
		fmt.Printf("[%s] "+format+"\n", append([]any{urlKey}, args...)...)
	}

	var room *roomAPI
	for {
		var err error
		room, err = fetchRoom(w.ctx, urlKey)
		if err == nil {
			break
		}
		var httpErr *httpStatusError
		if errors.As(err, &httpErr) && httpErr.Code/100 == 4 {
			fmt.Fprintf(os.Stderr, "[%s] Excluded: HTTP %d\n", urlKey, httpErr.Code)
			return
		}
		logf("Error: %s; retrying...", err)
		select {
		case <-w.ctx.Done():
			return
		case <-time.After(20 * time.Second):
		}
	}
	urlKey = room.URLKey
	if !w.claimWatch(urlKey) {
		// Another goroutine (started from a different campaign key) already
		// watches this room; stop rather than double-poll and double-record.
		if w.verbose {
			logf("Duplicate of an already-watched room; stopping")
		}
		return
	}
	if w.verbose {
		logf("%s", room.Name)
	}

	var prevSchedule int64
	// recentPassthrough is set when a Phase 1→2 passthrough download finds a
	// stream URL. It suppresses the subsequent is_live=true response that
	// SHOWROOM sets ~1 min after stream start, preventing a duplicate download
	// of the same session. Cleared when fetchRoom first returns is_live=false.
	var recentPassthrough bool
	if room.NextLiveSchedule != 0 {
		prevSchedule = room.NextLiveSchedule
		logf("Scheduled: %s", time.Unix(prevSchedule, 0).Format("2006-01-02 15:04:05"))
	}
	for {
		if room.IsLive {
			if !recentPassthrough {
				w.runDownload(urlKey, room)
			}
			room.IsLive = false
			room.NextLiveSchedule = 0
			prevSchedule = 0
		} else if room.NextLiveSchedule != 0 && time.Until(time.Unix(room.NextLiveSchedule, 0)) <= 0 {
			// Scheduled time has passed; poll stream URLs directly rather than
			// waiting for is_live, matching sba-stream Phase 1→2 transition.
			// resolveHLS has a 60 s timeout. After it returns (success or
			// timeout), NextLiveSchedule is cleared and fetchRoom runs; if the
			// API still reports a past schedule the passthrough re-fires then.
			if w.runDownload(urlKey, room) {
				recentPassthrough = true
			}
			room.NextLiveSchedule = 0
		}

		d := watchPollInterval(room.NextLiveSchedule)
		select {
		case <-w.ctx.Done():
			return
		case <-time.After(d):
		}

		// Scheduled time passed during the sleep: poll stream URLs before
		// fetchRoom (Phase 1→2 priority). NextLiveSchedule is cleared after
		// runDownload; fetchRoom then re-evaluates the live/schedule state.
		if room.NextLiveSchedule != 0 && time.Until(time.Unix(room.NextLiveSchedule, 0)) <= 0 {
			if w.runDownload(urlKey, room) {
				recentPassthrough = true
			}
			room.NextLiveSchedule = 0
		}

		updated, err := fetchRoom(w.ctx, urlKey)
		if err != nil {
			logf("Error: %s", err)
			select {
			case <-w.ctx.Done():
				return
			case <-time.After(20 * time.Second):
			}
			continue
		}

		if !updated.IsLive {
			recentPassthrough = false
		}

		if updated.NextLiveSchedule != 0 && updated.NextLiveSchedule != prevSchedule {
			prevSchedule = updated.NextLiveSchedule
			logf("Scheduled: %s", time.Unix(prevSchedule, 0).Format("2006-01-02 15:04:05"))
		}

		room = updated
	}
}

// runDownload resolves the HLS stream and records it with ffmpeg. It returns
// true when resolveHLS found a stream URL (the live event is considered
// consumed regardless of the ffmpeg outcome), and false when no stream was
// found or the context was cancelled before resolution.
func (w *watcher) runDownload(urlKey string, room *roomAPI) bool {
	if w.ctx.Err() != nil {
		return false
	}

	// Reserve the room before any work so a second trigger for the same
	// canonical url_key cannot start a concurrent recording of the same stream.
	// Held across resolveHLS + ffmpeg and released on return, so a genuine
	// resume after ffmpeg exits (stream still live) still proceeds on the next
	// poll.
	w.mu.Lock()
	if w.recording == nil {
		w.recording = make(map[string]bool)
	}
	if w.recording[urlKey] {
		w.mu.Unlock()
		return false
	}
	w.recording[urlKey] = true
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		delete(w.recording, urlKey)
		w.mu.Unlock()
	}()

	streamURL, err := w.resolveHLS(urlKey, room.ID)
	if err != nil {
		return false
	}

	start := time.Now()
	outPath := buildFileName(urlKey, start.Unix()) + ".mp4"

	proc, err := runner.StartFFmpeg(runner.FFmpegArgs{Input: streamURL, Detached: true}, outPath)
	if err != nil {
		fmt.Printf("[%s] Error: %s\n", urlKey, err)
		return true
	}

	fmt.Printf("[%s] File:      %s\n", urlKey, outPath)
	fmt.Printf("[%s] Recording: %s\n", urlKey, start.Format("2006-01-02 15:04:05"))

	w.mu.Lock()
	cancelled := w.ctx.Err() != nil
	w.active[urlKey] = proc // register before releasing so waitForDownloads tracks us
	w.mu.Unlock()

	if cancelled {
		// Shutdown fired between StartFFmpeg and registration; stop immediately.
		proc.Stop()
	}

	runErr := proc.Wait()

	w.mu.Lock()
	delete(w.active, urlKey)
	w.mu.Unlock()

	if runErr != nil {
		fmt.Printf("[%s] FFmpeg: %s\n", urlKey, runErr)
	}
	fmt.Printf("[%s] Finished: %s\n", urlKey, time.Now().Format("2006-01-02 15:04:05"))
	return true
}

// resolveHLS finds the best HLS URL for the room, with a 60s timeout so the
// watcher doesn't loop forever when a stream ends while the CDN still reports
// is_live=true. Returns a non-nil error (silently) on timeout or shutdown.
func (w *watcher) resolveHLS(urlKey string, roomID int) (string, error) {
	ctx, cancel := context.WithTimeout(w.ctx, 60*time.Second)
	defer cancel()

	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		api, err := fetchStreamingURLs(ctx, roomID)
		if err != nil {
			fmt.Printf("[%s] Error fetching streams: %s\n", urlKey, err)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(4 * time.Second):
			}
			continue
		}

		if best := selectBestHLS(api.StreamingURLList); best != nil {
			return best.URL, nil
		}

		fmt.Printf("[%s] No HLS stream; retrying...\n", urlKey)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(4 * time.Second):
		}
	}
}

func (w *watcher) stopAll() {
	w.mu.Lock()
	procs := make([]*runner.FFmpegProcess, 0, len(w.active))
	for _, proc := range w.active {
		procs = append(procs, proc)
	}
	w.mu.Unlock()
	for _, proc := range procs {
		proc.Stop()
	}
}

func (w *watcher) waitForDownloads() {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		w.mu.Lock()
		n := len(w.active)
		w.mu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Timeout: force-kill any remaining processes.
	w.mu.Lock()
	procs := make([]*runner.FFmpegProcess, 0, len(w.active))
	for _, proc := range w.active {
		procs = append(procs, proc)
	}
	w.mu.Unlock()
	for _, proc := range procs {
		proc.Kill()
	}
}

// defaultPollInterval is the idle poll interval (no schedule). Overridden in tests.
var defaultPollInterval = 20 * time.Second

func watchPollInterval(nextSchedule int64) time.Duration {
	if nextSchedule == 0 {
		return defaultPollInterval
	}
	remaining := time.Until(time.Unix(nextSchedule, 0))
	switch {
	case remaining > 0:
		return min(remaining, 20*time.Second)
	case -remaining < 5*time.Minute:
		return 4 * time.Second
	case -remaining < 20*time.Minute:
		return 8 * time.Second
	default:
		return 20 * time.Second
	}
}
