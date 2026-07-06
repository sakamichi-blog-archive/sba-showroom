package showroom

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const sampleEpisodeHTML = `<!DOCTYPE html>
<html>
<body>
<div id="episode-data"
     data-episode="{&quot;id&quot;:&quot;14&quot;,&quot;title&quot;:&quot;Test Episode&quot;,&quot;description&quot;:&quot;&quot;,&quot;display_started_at&quot;:1700000000,&quot;display_ended_at&quot;:null,&quot;display_status&quot;:2,&quot;video_id&quot;:99,&quot;video_time&quot;:3600,&quot;series&quot;:{&quot;id&quot;:1,&quot;name&quot;:&quot;Test Series&quot;,&quot;description&quot;:&quot;&quot;,&quot;display_status&quot;:2},&quot;thumbnail_url&quot;:&quot;&quot;}">
</div>
</body>
</html>`

func TestParseEpisodeData(t *testing.T) {
	ep, err := parseEpisodeData(sampleEpisodeHTML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.Title != "Test Episode" {
		t.Errorf("Title: got %q, want %q", ep.Title, "Test Episode")
	}
	if ep.Series.Name != "Test Series" {
		t.Errorf("Series.Name: got %q, want %q", ep.Series.Name, "Test Series")
	}
	if ep.DisplayStartedAt != 1700000000 {
		t.Errorf("DisplayStartedAt: got %d, want 1700000000", ep.DisplayStartedAt)
	}
}

func TestParseEpisodeData_MissingDiv(t *testing.T) {
	_, err := parseEpisodeData(`<html><body><p>no data here</p></body></html>`)
	if err == nil {
		t.Fatal("expected error for missing episode-data div, got nil")
	}
}

func TestExtractCloudfrontCookies(t *testing.T) {
	headers := []string{
		"CloudFront-Key-Pair-Id=APKA; Path=/; Secure",
		"CloudFront-Policy=abc123; Path=/; Secure",
		"session=xyz; Path=/",
		"CloudFront-Signature=sig; Path=/; Secure",
	}
	got := extractCloudfrontCookies(headers)
	want := "CloudFront-Key-Pair-Id=APKA; CloudFront-Policy=abc123; CloudFront-Signature=sig"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExtractCloudfrontCookies_None(t *testing.T) {
	got := extractCloudfrontCookies([]string{"session=abc; Path=/"})
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFetchEpisode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/episode/watch":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(sampleEpisodeHTML))
		case "/api/episode/streaming_url":
			w.Header().Add("Set-Cookie", "CloudFront-Key-Pair-Id=APKA; Path=/; Secure")
			w.Header().Add("Set-Cookie", "CloudFront-Signature=sig; Path=/; Secure")
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"streaming_url_list": {
					"hls_all": {"hls_all": "https://cdn.example.com/stream/all.m3u8", "quality": "all"},
					"hls_low": {"hls": "https://cdn.example.com/stream/low.m3u8", "quality": "low"},
					"hls_medium": {"hls": "https://cdn.example.com/stream/med.m3u8", "quality": "medium"},
					"hls_source": {"hls": "https://cdn.example.com/stream/src.m3u8", "quality": "source"}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	old := showroomBaseURL
	showroomBaseURL = srv.URL
	defer func() { showroomBaseURL = old }()

	ep, err := fetchEpisode(srv.URL+"/episode/watch?id=14", "14")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.title != "Test Episode" {
		t.Errorf("title: got %q, want %q", ep.title, "Test Episode")
	}
	if ep.series != "Test Series" {
		t.Errorf("series: got %q, want %q", ep.series, "Test Series")
	}
	if ep.streamURL != "https://cdn.example.com/stream/all.m3u8" {
		t.Errorf("streamURL: got %q", ep.streamURL)
	}
	if ep.cookies != "CloudFront-Key-Pair-Id=APKA; CloudFront-Signature=sig" {
		t.Errorf("cookies: got %q", ep.cookies)
	}
	if ep.startedAt != time.Unix(1700000000, 0) {
		t.Errorf("startedAt: got %v", ep.startedAt)
	}
}
