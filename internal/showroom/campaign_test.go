package showroom

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestFetchCampaignRooms(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groupA":["room_a1","room_a2"],"groupB":["room_b1"]}`))
	}))
	defer srv.Close()

	old := campaignRoomsURL["nogi"]
	campaignRoomsURL["nogi"] = srv.URL
	defer func() { campaignRoomsURL["nogi"] = old }()

	rooms, err := fetchCampaignRooms("nogi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rooms) != 3 {
		t.Errorf("got %d rooms, want 3", len(rooms))
	}
	sort.Strings(rooms)
	for i, want := range []string{"room_a1", "room_a2", "room_b1"} {
		if rooms[i] != want {
			t.Errorf("rooms[%d] = %q, want %q", i, rooms[i], want)
		}
	}
}

func TestFetchCampaignRooms_UnknownCampaign(t *testing.T) {
	_, err := fetchCampaignRooms("unknown")
	if err == nil {
		t.Fatal("expected error for unknown campaign slug")
	}
}

func TestFetchCampaignRooms_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusGone)
	}))
	defer srv.Close()

	old := campaignRoomsURL["nogi"]
	campaignRoomsURL["nogi"] = srv.URL
	defer func() { campaignRoomsURL["nogi"] = old }()

	_, err := fetchCampaignRooms("nogi")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}
