package showroom

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/sakamichi-blog-archive/sba-showroom/internal/runner"
)

var roomURLRegex = regexp.MustCompile(`(?i)^https://www\.showroom-live\.com/(r/([-_0-9A-Za-z]+)|([-_0-9A-Za-z]+))$`)

type DownloadOptions struct {
	URL          string
	Retry        bool
	ExpectedTime *time.Time
}

func Download(opts DownloadOptions) error {
	u, err := url.Parse(opts.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	urlNoQuery := *u
	urlNoQuery.RawQuery = ""
	return downloadLive(opts, urlNoQuery.String())
}

func downloadLive(opts DownloadOptions, urlNoQuery string) error {
	matches := roomURLRegex.FindStringSubmatch(urlNoQuery)
	if matches == nil {
		return fmt.Errorf("URL does not match a supported SHOWROOM format")
	}
	roomURLKey := matches[2] // r/KEY form
	if roomURLKey == "" {
		roomURLKey = matches[3]
	}

	fmt.Printf("Fetching room: %s\n", roomURLKey)
	room, err := fetchRoom(roomURLKey)
	if err != nil {
		return fmt.Errorf("fetch room: %w", err)
	}
	fmt.Printf("Name: %s\n", room.Name)

	// Use canonical url_key from API.
	roomURLKey = room.URLKey

	var expectedTS int64
	if opts.ExpectedTime != nil {
		expectedTS = opts.ExpectedTime.Unix()
	}

	if !room.IsLive {
		if err := waitForLive(room, roomURLKey, &expectedTS); err != nil {
			return err
		}
		room, err = fetchRoom(roomURLKey)
		if err != nil {
			return fmt.Errorf("re-fetch room: %w", err)
		}
	}

	return runDownloadLoop(opts, room, roomURLKey, expectedTS)
}

func waitForLive(room *roomAPI, roomURLKey string, expectedTS *int64) error {
	if room.NextLiveSchedule != 0 {
		*expectedTS = room.NextLiveSchedule
		fmt.Printf("Status:    Scheduled\n")
		fmt.Printf("Scheduled: %s\n", time.Unix(*expectedTS, 0).Format("2006-01-02 15:04:05"))
	} else if *expectedTS != 0 {
		fmt.Printf("Expect: %s\n", time.Unix(*expectedTS, 0).Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("Status: Not live\n")
	}

	for {
		var sleep time.Duration
		if *expectedTS == 0 {
			sleep = 20 * time.Second
		} else {
			remaining := time.Until(time.Unix(*expectedTS, 0))
			switch {
			case remaining > 0:
				sleep = 20 * time.Second
			case -remaining < 5*time.Minute:
				sleep = 4 * time.Second
			case -remaining < 20*time.Minute:
				sleep = 8 * time.Second
			default:
				sleep = 20 * time.Second
			}
		}

		if sleep > 0 {
			time.Sleep(sleep)
		}

		updated, err := fetchRoom(roomURLKey)
		if err != nil {
			fmt.Printf("  Error: %s\n", err)
			time.Sleep(20 * time.Second)
			continue
		}
		if updated.IsLive {
			*room = *updated
			return nil
		}
		if updated.NextLiveSchedule != 0 && updated.NextLiveSchedule != *expectedTS {
			*expectedTS = updated.NextLiveSchedule
			fmt.Printf("Scheduled: %s\n", time.Unix(*expectedTS, 0).Format("2006-01-02 15:04:05"))
		}
	}
}

func runDownloadLoop(opts DownloadOptions, room *roomAPI, roomURLKey string, expectedTS int64) error {
	for {
		streamURL, err := resolveHLSURL(room.ID)
		if err != nil {
			return err
		}

		ts := expectedTS
		if ts == 0 {
			ts = time.Now().Unix()
		}
		outPath := buildFileName(roomURLKey, ts) + ".mp4"

		fmt.Printf("Status:    Live\n")
		fmt.Printf("File:      %s\n", outPath)
		fmt.Printf("Recording: %s\n", time.Now().Format("2006-01-02 15:04:05"))

		runErr := runner.FFmpeg(runner.FFmpegArgs{Input: streamURL}, outPath)

		fmt.Printf("Finished:  %s\n", time.Now().Format("2006-01-02 15:04:05"))

		// In retry mode the loop runs until killed, so exit code is not meaningful.
		if !opts.Retry {
			return noRetryOutcome(runErr, outPath)
		}
		if runErr != nil {
			fmt.Printf("Error: %s — retrying...\n", runErr)
		} else {
			fmt.Printf("Retrying...\n")
		}

		updated, err := fetchRoom(roomURLKey)
		if err == nil {
			room = updated
		}
	}
}

// noRetryOutcome returns nil (exit 0) if the output file has content, even when
// ffmpeg errors — streams often terminate abruptly with a non-zero exit despite
// producing a valid recording. Returns an error when no content was written,
// regardless of ffmpeg's exit code.
func noRetryOutcome(runErr error, outPath string) error {
	if fileHasContent(outPath) {
		if runErr != nil {
			fmt.Printf("Warning: %s\n", runErr)
		}
		return nil
	}
	if runErr != nil {
		return runErr
	}
	return fmt.Errorf("ffmpeg exited successfully but no output was written to %s", outPath)
}

func fileHasContent(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular() && fi.Size() > 0
}

func resolveHLSURL(roomID int) (string, error) {
	for {
		api, err := fetchStreamingURLs(roomID)
		if err != nil {
			fmt.Printf("  Error fetching streams: %s\n", err)
			time.Sleep(4 * time.Second)
			continue
		}

		var best *streamingURLItem
		for i := range api.StreamingURLList {
			item := &api.StreamingURLList[i]
			if item.Type == "hls" && (best == nil || item.Quality > best.Quality) {
				best = item
			}
		}
		if best != nil {
			return best.URL, nil
		}

		fmt.Println("  No HLS streams found; retrying...")
		time.Sleep(4 * time.Second)
	}
}

var jst = time.FixedZone("JST", 9*60*60)

func buildFileName(name string, unixTS int64) string {
	date := time.Unix(unixTS, 0).In(jst).Format("060102")
	safe := sanitizeName(name)
	return date + "-" + safe + "-" + randomHex(4)
}

func sanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func randomHex(n int) string {
	buf := make([]byte, (n+1)/2)
	rand.Read(buf)
	return hex.EncodeToString(buf)[:n]
}
