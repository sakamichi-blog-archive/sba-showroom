package showroom

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRoom(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/room/46_iwamotorenka" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":123,"is_live":true,"name":"Renka Iwamoto","next_live_schedule":0,"url_key":"46_iwamotorenka"}`))
	}))
	defer srv.Close()

	old := cdnBaseURL
	cdnBaseURL = srv.URL
	defer func() { cdnBaseURL = old }()

	room, err := fetchRoom("46_iwamotorenka")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.ID != 123 {
		t.Errorf("ID: got %d, want 123", room.ID)
	}
	if room.Name != "Renka Iwamoto" {
		t.Errorf("Name: got %q, want %q", room.Name, "Renka Iwamoto")
	}
	if !room.IsLive {
		t.Error("IsLive: got false, want true")
	}
}

func TestFetchRoom_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	old := cdnBaseURL
	cdnBaseURL = srv.URL
	defer func() { cdnBaseURL = old }()

	_, err := fetchRoom("nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFetchRoom_NonOK_IsHTTPStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	old := cdnBaseURL
	cdnBaseURL = srv.URL
	defer func() { cdnBaseURL = old }()

	_, err := fetchRoom("someroom")
	var httpErr *httpStatusError
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected httpStatusError, got %T: %v", err, err)
	}
	if httpErr.Code != http.StatusNotFound {
		t.Errorf("Code: got %d, want %d", httpErr.Code, http.StatusNotFound)
	}
}

func TestFetchRoom_WrongContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html></html>`))
	}))
	defer srv.Close()

	old := cdnBaseURL
	cdnBaseURL = srv.URL
	defer func() { cdnBaseURL = old }()

	_, err := fetchRoom("someroom")
	if err == nil {
		t.Fatal("expected error for wrong Content-Type, got nil")
	}
}

func TestFetchStreamingURLs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"streaming_url_list": [
				{"type":"hls","quality":100,"url":"https://example.com/high.m3u8","is_default":true},
				{"type":"hls","quality":10,"url":"https://example.com/low.m3u8","is_default":false}
			]
		}`))
	}))
	defer srv.Close()

	old := showroomBaseURL
	showroomBaseURL = srv.URL
	defer func() { showroomBaseURL = old }()

	api, err := fetchStreamingURLs(123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(api.StreamingURLList) != 2 {
		t.Errorf("StreamingURLList length: got %d, want 2", len(api.StreamingURLList))
	}
}
