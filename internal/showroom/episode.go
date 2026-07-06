package showroom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type episodeStream struct {
	title     string
	series    string
	streamURL string
	cookies   string
	startedAt time.Time
}

func fetchEpisode(rawURL, episodeID string) (*episodeStream, error) {
	req, err := http.NewRequest("GET", "https://www.showroom-live.com/episode/watch?id="+episodeID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("episode page returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ep, err := parseEpisodeData(string(body))
	if err != nil {
		return nil, err
	}

	streamResp, err := fetchEpisodeStreamingURL(rawURL, episodeID)
	if err != nil {
		return nil, err
	}

	cookies := extractCloudfrontCookies(streamResp.Header.Values("Set-Cookie"))

	var streamAPI episodeStreamingURLAPI
	streamBody, err := io.ReadAll(streamResp.Body)
	streamResp.Body.Close()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(streamBody, &streamAPI); err != nil {
		return nil, fmt.Errorf("parse episode streaming URL response: %w", err)
	}

	return &episodeStream{
		title:     ep.Title,
		series:    ep.Series.Name,
		streamURL: streamAPI.StreamingURLList.HLSAll.HLSAll,
		cookies:   cookies,
		startedAt: time.Unix(ep.DisplayStartedAt, 0),
	}, nil
}

func parseEpisodeData(htmlBody string) (*episodeData, error) {
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return nil, err
	}

	attr := findAttr(doc, "div", "id", "episode-data", "data-episode")
	if attr == "" {
		return nil, fmt.Errorf("could not find episode data in page HTML")
	}

	var ep episodeData
	if err := json.Unmarshal([]byte(attr), &ep); err != nil {
		return nil, fmt.Errorf("parse episode data JSON: %w", err)
	}
	return &ep, nil
}

func findAttr(n *html.Node, tag, keyAttr, keyVal, targetAttr string) string {
	if n.Type == html.ElementNode && n.Data == tag {
		var hasKey bool
		var target string
		for _, a := range n.Attr {
			if a.Key == keyAttr && a.Val == keyVal {
				hasKey = true
			}
			if a.Key == targetAttr {
				target = a.Val
			}
		}
		if hasKey {
			return target
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := findAttr(c, tag, keyAttr, keyVal, targetAttr); v != "" {
			return v
		}
	}
	return ""
}

func fetchEpisodeStreamingURL(referer, episodeID string) (*http.Response, error) {
	req, err := http.NewRequest("GET",
		"https://www.showroom-live.com/api/episode/streaming_url?episode_id="+episodeID,
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", referer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("episode streaming URL API returned HTTP %d", resp.StatusCode)
	}
	return resp, nil
}

func extractCloudfrontCookies(setCookieHeaders []string) string {
	var parts []string
	for _, raw := range setCookieHeaders {
		// Each Set-Cookie value: "name=value; Path=...; ..."
		nameVal := strings.SplitN(raw, ";", 2)[0]
		name := strings.SplitN(nameVal, "=", 2)[0]
		if strings.HasPrefix(strings.ToLower(name), "cloudfront-") {
			parts = append(parts, nameVal)
		}
	}
	return strings.Join(parts, "; ")
}
