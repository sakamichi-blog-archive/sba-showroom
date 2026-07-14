package showroom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var campaignRoomsURL = map[string]string{
	"nogi":   "https://campaign.showroom-live.com/nogizaka46_sr/data/rooms.json",
	"hinata": "https://campaign.showroom-live.com/hinatazaka46_sr/data/rooms.json",
	"sakura": "https://campaign.showroom-live.com/sakurazaka46_sr/data/rooms.json",
}

// fetchCampaignRooms returns all room URL keys for a campaign slug.
// rooms.json groups rooms by generation under varying keys, so we collect all
// string values regardless of key name.
func fetchCampaignRooms(ctx context.Context, campaign string) ([]string, error) {
	url, ok := campaignRoomsURL[campaign]
	if !ok {
		return nil, fmt.Errorf("unknown campaign %q (valid: nogi, hinata, sakura)", campaign)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var groups map[string][]string
	if err := json.Unmarshal(body, &groups); err != nil {
		return nil, fmt.Errorf("parse rooms.json: %w", err)
	}

	var rooms []string
	for _, keys := range groups {
		for _, k := range keys {
			if k != "" {
				rooms = append(rooms, k)
			}
		}
	}
	return rooms, nil
}
