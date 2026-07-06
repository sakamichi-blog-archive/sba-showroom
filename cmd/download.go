package cmd

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sakamichi-blog-archive/sba-showroom/internal/showroom"
)

func runDownload(args []string) {
	fs := flag.NewFlagSet("download", flag.ExitOnError)
	hls := fs.Bool("hls", false, "Prefer HLS over RTMP")
	noRetry := fs.Bool("no-retry", false, "Stop after stream ends (retry is on by default)")
	fs.Usage = func() {
		fmt.Println("Usage: sba-showroom download [flags] URL [EXPECTED_TIME]")
		fmt.Println()
		fmt.Println("URL formats:")
		fmt.Println("  https://www.showroom-live.com/ROOM_URL_KEY")
		fmt.Println("  https://www.showroom-live.com/r/ROOM_URL_KEY")
		fmt.Println()
		fmt.Println("EXPECTED_TIME formats:")
		fmt.Println("  HH:mm")
		fmt.Println("  YYYY-MM-DD HH:mm")
		fmt.Println("  YYYY/MM/DD HH:mm")
		fmt.Println()
		fmt.Println("Flags:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	positional := fs.Args()
	if len(positional) < 1 {
		fs.Usage()
		os.Exit(1)
	}

	rawURL := positional[0]
	var expectedTime *time.Time
	if len(positional) >= 2 {
		t, err := parseExpectedTime(positional[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid expected time %q: %v\n", positional[1], err)
			os.Exit(1)
		}
		expectedTime = &t
	}

	opts := showroom.DownloadOptions{
		URL:          rawURL,
		PreferHLS:    *hls,
		Retry:        !*noRetry,
		ExpectedTime: expectedTime,
	}
	if err := showroom.Download(opts); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

var timeFormats = []string{
	"2006-01-02 15:04",
	"2006/01/02 15:04",
	"15:04",
}

func parseExpectedTime(s string) (time.Time, error) {
	now := time.Now()
	for _, layout := range timeFormats {
		if layout == "15:04" {
			t, err := time.ParseInLocation("15:04", s, time.Local)
			if err == nil {
				return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
			}
			continue
		}
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("none of the supported formats matched")
}
