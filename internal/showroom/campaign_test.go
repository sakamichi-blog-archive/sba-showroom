package showroom

import (
	"context"
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

	rooms, err := fetchCampaignRooms(context.Background(), "nogi")
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

func TestFetchCampaignRooms_FiltersEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groupA":["room_a1","","room_a2"]}`))
	}))
	defer srv.Close()

	old := campaignRoomsURL["nogi"]
	campaignRoomsURL["nogi"] = srv.URL
	defer func() { campaignRoomsURL["nogi"] = old }()

	rooms, err := fetchCampaignRooms(context.Background(), "nogi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range rooms {
		if r == "" {
			t.Error("expected empty strings to be filtered out")
		}
	}
	if len(rooms) != 2 {
		t.Errorf("got %d rooms, want 2", len(rooms))
	}
}

func TestFetchCampaignRooms_UnknownCampaign(t *testing.T) {
	_, err := fetchCampaignRooms(context.Background(), "unknown")
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

	_, err := fetchCampaignRooms(context.Background(), "nogi")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}
