package cmd

import (
	"fmt"
	"time"

	"github.com/sakamichi-blog-archive/sba-showroom/internal/showroom"
	"github.com/spf13/cobra"
)

var (
	flagHLS     bool
	flagNoRetry bool
)

var downloadCmd = &cobra.Command{
	Use:   "download URL [EXPECTED_TIME]",
	Short: "Download a SHOWROOM livestream or episode",
	Long: `Download a SHOWROOM livestream or recorded episode.

URL formats:
  https://www.showroom-live.com/ROOM_URL_KEY
  https://www.showroom-live.com/r/ROOM_URL_KEY

EXPECTED_TIME formats (for livestreams):
  HH:mm
  YYYY-MM-DD HH:mm
  YYYY/MM/DD HH:mm`,
	Example: `  sba-showroom download https://www.showroom-live.com/46_iwamotorenka
  sba-showroom download https://www.showroom-live.com/r/46_iwamotorenka
  sba-showroom download https://www.showroom-live.com/46_iwamotorenka 17:30 --no-retry`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runDownload,
}

func init() {
	downloadCmd.Flags().BoolVar(&flagHLS, "hls", false, "Prefer HLS over RTMP")
	downloadCmd.Flags().BoolVar(&flagNoRetry, "no-retry", false, "Stop after stream ends")
}

func runDownload(cmd *cobra.Command, args []string) error {
	rawURL := args[0]

	var expectedTime *time.Time
	if len(args) == 2 {
		t, err := parseExpectedTime(args[1])
		if err != nil {
			return fmt.Errorf("invalid expected time %q: %w", args[1], err)
		}
		expectedTime = &t
	}

	opts := showroom.DownloadOptions{
		URL:          rawURL,
		PreferHLS:    flagHLS,
		Retry:        !flagNoRetry,
		ExpectedTime: expectedTime,
	}
	return showroom.Download(opts)
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
				// Combine today's date with parsed time.
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
