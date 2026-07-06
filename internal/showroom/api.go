package showroom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.181 Safari/537.36"

var (
	cdnBaseURL      = "https://public-api.showroom-cdn.com"
	showroomBaseURL = "https://www.showroom-live.com"
)

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
	IsDefault  bool   `json:"is_default"`
	Quality    int    `json:"quality"`
	StreamName string `json:"stream_name"`
	Type       string `json:"type"`
	URL        string `json:"url"`
}


func getJSON(url, referer string, out interface{}) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("HTTP %d", resp.StatusCode)
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

func fetchRoom(roomURLKey string) (*roomAPI, error) {
	var room roomAPI
	_, err := getJSON(cdnBaseURL+"/room/"+roomURLKey, "", &room)
	return &room, err
}

func fetchStreamingURLs(roomID int) (*streamingURLAPI, error) {
	url := fmt.Sprintf(
		showroomBaseURL+"/api/live/streaming_url?room_id=%d&ignore_low_stream=1",
		roomID,
	)
	var api streamingURLAPI
	_, err := getJSON(url, "", &api)
	return &api, err
}
