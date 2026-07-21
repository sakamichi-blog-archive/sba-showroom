package showroom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.181 Safari/537.36"

var (
	cdnBaseURL      = "https://public-api.showroom-cdn.com"
	showroomBaseURL = "https://www.showroom-live.com"
	httpClient      = &http.Client{Timeout: 30 * time.Second}
)

type httpStatusError struct {
	Code int
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("HTTP %d", e.Code)
}

type roomAPI struct {
	ID               int    `json:"id"`
	IsLive           bool   `json:"is_live"`
	Name             string `json:"name"`
	NextLiveSchedule int64  `json:"next_live_schedule"`
	URLKey           string `json:"url_key"`
}

type streamingURLAPI struct {
	StreamingURLList []streamingURLItem `json:"streaming_url_list"`
}

type streamingURLItem struct {
	IsDefault bool   `json:"is_default"`
	Quality   int    `json:"quality"`
	Type      string `json:"type"`
	URL       string `json:"url"`
}

func getJSON(ctx context.Context, url, referer string, out interface{}) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return resp, &httpStatusError{Code: resp.StatusCode}
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		return resp, fmt.Errorf("unexpected Content-Type: %s", ct)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	return resp, json.Unmarshal(body, out)
}

func fetchRoom(ctx context.Context, roomURLKey string) (*roomAPI, error) {
	var room roomAPI
	_, err := getJSON(ctx, cdnBaseURL+"/room/"+roomURLKey, "", &room)
	return &room, err
}

func fetchStreamingURLs(ctx context.Context, roomID int) (*streamingURLAPI, error) {
	url := fmt.Sprintf(
		showroomBaseURL+"/api/live/streaming_url?room_id=%d&ignore_low_stream=1&_=%d",
		roomID, time.Now().Unix(),
	)
	var api streamingURLAPI
	_, err := getJSON(ctx, url, "", &api)
	return &api, err
}

// selectBestHLS returns the highest-quality HLS stream from the list, or nil if none exist.
func selectBestHLS(items []streamingURLItem) *streamingURLItem {
	var best *streamingURLItem
	for i := range items {
		item := &items[i]
		if item.Type == "hls" && (best == nil || item.Quality > best.Quality) {
			best = item
		}
	}
	return best
}
