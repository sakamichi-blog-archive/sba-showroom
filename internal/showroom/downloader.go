package showroom

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sakamichi-blog-archive/sba-showroom/internal/runner"
)

var roomURLRegex = regexp.MustCompile(`(?i)^https://www\.showroom-live\.com/(r/([-_0-9A-Za-z]+)|([-_0-9A-Za-z]+))$`)

type DownloadOptions struct {
	URL          string
	PreferHLS    bool
	Retry        bool
	ExpectedTime *time.Time
}

func Download(opts DownloadOptions) error {
	u, err := url.Parse(opts.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Strip query string for regex matching.
	urlNoQuery := *u
	urlNoQuery.RawQuery = ""

	if u.Path == "/episode/watch" {
		return downloadEpisode(opts)
	}

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
		// Re-fetch room after standby.
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
			case remaining > 4*time.Minute:
				sleep = 30 * time.Second
			case remaining > 0:
				sleep = 20 * time.Second
			default:
				// Past expected time; check immediately.
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
	for attempt := 0; ; attempt++ {
		streamURL, streamType, err := resolveStreamURL(room.ID, opts.PreferHLS)
		if err != nil {
			return err
		}

		ts := expectedTS
		if ts == 0 {
			ts = time.Now().Unix()
		}
		fileName := buildFileName(roomURLKey, ts) + ".mp4"
		outPath := fileName

		fmt.Printf("Status:    Live\n")
		fmt.Printf("File:      %s\n", outPath)
		fmt.Printf("Recording: %s\n", time.Now().Format("2006-01-02 15:04:05"))

		var ffArgs []runner.FFmpegArgs
		switch streamType {
		case "rtmp":
			ffArgs = []runner.FFmpegArgs{{Input: streamURL}}
		case "hls":
			ffArgs = []runner.FFmpegArgs{{Input: streamURL}}
		}

		runErr := runner.FFmpeg(ffArgs[0], outPath)

		fmt.Printf("Finished:  %s\n", time.Now().Format("2006-01-02 15:04:05"))

		if !opts.Retry {
			return runErr
		}
		if runErr != nil {
			fmt.Printf("Error: %s — retrying...\n", runErr)
		} else {
			fmt.Printf("Retrying...\n")
		}

		// Re-fetch room state for the next attempt.
		updated, err := fetchRoom(roomURLKey)
		if err == nil {
			room = updated
		}
	}
}

func resolveStreamURL(roomID int, preferHLS bool) (streamURL, streamType string, err error) {
	for {
		api, fetchErr := fetchStreamingURLs(roomID)
		if fetchErr != nil {
			fmt.Printf("  Error fetching streams: %s\n", fetchErr)
			time.Sleep(4 * time.Second)
			continue
		}

		u, t, ok := selectStream(api.StreamingURLList, preferHLS)
		if ok {
			return u, t, nil
		}

		fmt.Println("  No streams found; retrying...")
		time.Sleep(4 * time.Second)
	}
}

func selectStream(items []streamingURLItem, preferHLS bool) (streamURL, streamType string, ok bool) {
	var rtmpItems, hlsItems []streamingURLItem
	for _, item := range items {
		switch item.Type {
		case "rtmp":
			rtmpItems = append(rtmpItems, item)
		case "hls":
			hlsItems = append(hlsItems, item)
		}
	}
	sort.Slice(rtmpItems, func(i, j int) bool { return rtmpItems[i].Quality > rtmpItems[j].Quality })
	sort.Slice(hlsItems, func(i, j int) bool { return hlsItems[i].Quality > hlsItems[j].Quality })

	if preferHLS {
		if len(hlsItems) > 0 {
			return hlsItems[0].URL, "hls", true
		}
		if len(rtmpItems) > 0 {
			fmt.Println("  No HLS streams available. Using RTMP...")
			return rtmpItems[0].URL + "/" + rtmpItems[0].StreamName, "rtmp", true
		}
	} else {
		if len(rtmpItems) > 0 {
			return rtmpItems[0].URL + "/" + rtmpItems[0].StreamName, "rtmp", true
		}
		if len(hlsItems) > 0 {
			fmt.Println("  No RTMP streams available. Using HLS...")
			return hlsItems[0].URL, "hls", true
		}
	}
	return "", "", false
}

func downloadEpisode(opts DownloadOptions) error {
	u, _ := url.Parse(opts.URL)
	episodeID := u.Query().Get("id")
	if episodeID == "" {
		return fmt.Errorf("missing episode ID in URL")
	}

	fmt.Printf("Fetching episode %s...\n", episodeID)
	ep, err := fetchEpisode(opts.URL, episodeID)
	if err != nil {
		return fmt.Errorf("fetch episode: %w", err)
	}

	fmt.Printf("Series:  %s\n", ep.series)
	fmt.Printf("Episode: %s\n", ep.title)

	ts := ep.startedAt.Unix()
	fileName := buildFileName(ep.title, ts) + ".mp4"
	outPath := fileName

	fmt.Printf("File:      %s\n", outPath)
	fmt.Printf("Recording: %s\n", time.Now().Format("2006-01-02 15:04:05"))

	args := runner.FFmpegArgs{
		Input:   ep.streamURL,
		Headers: map[string]string{"Cookie": ep.cookies},
	}
	if err := runner.FFmpeg(args, outPath); err != nil {
		return fmt.Errorf("ffmpeg: %w", err)
	}

	// Set file mtime to stream start time.
	_ = os.Chtimes(outPath, ep.startedAt, ep.startedAt)

	fmt.Printf("Finished:  %s\n", time.Now().Format("2006-01-02 15:04:05"))
	return nil
}

func buildFileName(name string, unixTS int64) string {
	date := time.Unix(unixTS, 0).Format("060102")
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
