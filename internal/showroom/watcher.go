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
	active  map[string]*runner.FFmpegProcess // urlKey → in-progress download
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
		ctx:     ctx,
		cancel:  cancel,
		verbose: opts.Verbose,
		active:  make(map[string]*runner.FFmpegProcess),
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
	if w.verbose {
		logf("%s", room.Name)
	}

	var prevSchedule int64
	if room.NextLiveSchedule != 0 {
		prevSchedule = room.NextLiveSchedule
		logf("Scheduled: %s", time.Unix(prevSchedule, 0).Format("2006-01-02 15:04:05"))
	}
	for {
		if room.IsLive {
			w.runDownload(urlKey, room)
			room.IsLive = false
			room.NextLiveSchedule = 0
			prevSchedule = 0
		}

		d := watchPollInterval(room.NextLiveSchedule)
		select {
		case <-w.ctx.Done():
			return
		case <-time.After(d):
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

		if updated.NextLiveSchedule != 0 && updated.NextLiveSchedule != prevSchedule {
			prevSchedule = updated.NextLiveSchedule
			logf("Scheduled: %s", time.Unix(prevSchedule, 0).Format("2006-01-02 15:04:05"))
		}

		room = updated
	}
}

func (w *watcher) runDownload(urlKey string, room *roomAPI) {
	if w.ctx.Err() != nil {
		return
	}

	streamURL, err := w.resolveHLS(urlKey, room.ID)
	if err != nil {
		return
	}

	outPath := buildFileName(urlKey, time.Now().Unix()) + ".mp4"
	fmt.Printf("[%s] Recording: %s\n", urlKey, outPath)

	proc, err := runner.StartFFmpeg(runner.FFmpegArgs{Input: streamURL, Detached: true}, outPath)
	if err != nil {
		fmt.Printf("[%s] Error: %s\n", urlKey, err)
		return
	}

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
	fmt.Printf("[%s] Finished: %s\n", urlKey, outPath)
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

func watchPollInterval(nextSchedule int64) time.Duration {
	if nextSchedule == 0 {
		return 20 * time.Second
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
