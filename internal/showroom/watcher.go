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

type watcher struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	active map[string]*runner.FFmpegProcess // urlKey → in-progress download
}

// Watch fetches rooms for each campaign slug and monitors them for live streams,
// starting concurrent downloads as rooms go live.
func Watch(campaignSlugs []string) error {
	seen := make(map[string]bool)
	var roomKeys []string
	for _, slug := range campaignSlugs {
		keys, err := fetchCampaignRooms(slug)
		if err != nil {
			return fmt.Errorf("campaign %s: %w", slug, err)
		}
		for _, k := range keys {
			if !seen[k] {
				seen[k] = true
				roomKeys = append(roomKeys, k)
			}
		}
	}

	if len(roomKeys) == 1 {
		fmt.Printf("Watching %s...\n", roomKeys[0])
	} else {
		fmt.Printf("Watching %s and %d other rooms...\n", roomKeys[0], len(roomKeys)-1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := &watcher{
		ctx:    ctx,
		cancel: cancel,
		active: make(map[string]*runner.FFmpegProcess),
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
			w.cancel()
			os.Exit(0)
		}

		now := time.Now()
		if !firstInterrupt.IsZero() && now.Sub(firstInterrupt) < 3*time.Second {
			fmt.Printf("\nStopping %d download(s)...\n", n)
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

	room, err := fetchRoom(urlKey)
	if err != nil {
		var httpErr *httpStatusError
		if !errors.As(err, &httpErr) {
			logf("Error: %s", err)
		}
		return
	}
	urlKey = room.URLKey

	var prevSchedule int64
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

		updated, err := fetchRoom(urlKey)
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

	streamURL, err := resolveHLSURL(room.ID)
	if err != nil {
		fmt.Printf("[%s] Error resolving stream: %s\n", urlKey, err)
		return
	}

	outPath := buildFileName(urlKey, time.Now().Unix()) + ".mp4"
	fmt.Printf("[%s] Recording: %s\n", urlKey, outPath)

	proc, err := runner.StartFFmpeg(runner.FFmpegArgs{Input: streamURL}, outPath)
	if err != nil {
		fmt.Printf("[%s] Error: %s\n", urlKey, err)
		return
	}

	w.mu.Lock()
	w.active[urlKey] = proc
	w.mu.Unlock()

	runErr := proc.Wait()

	w.mu.Lock()
	delete(w.active, urlKey)
	w.mu.Unlock()

	if runErr != nil {
		fmt.Printf("[%s] FFmpeg: %s\n", urlKey, runErr)
	}
	fmt.Printf("[%s] Finished: %s\n", urlKey, outPath)
}

func (w *watcher) stopAll() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, proc := range w.active {
		proc.Stop()
	}
}

func (w *watcher) waitForDownloads() {
	for {
		w.mu.Lock()
		n := len(w.active)
		w.mu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func watchPollInterval(nextSchedule int64) time.Duration {
	if nextSchedule == 0 {
		return 20 * time.Second
	}
	remaining := time.Until(time.Unix(nextSchedule, 0))
	switch {
	case remaining > 0:
		return 20 * time.Second
	case -remaining < 5*time.Minute:
		return 4 * time.Second
	case -remaining < 20*time.Minute:
		return 8 * time.Second
	default:
		return 20 * time.Second
	}
}
