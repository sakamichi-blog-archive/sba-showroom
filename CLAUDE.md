# CLAUDE.md

## Project

Go implementation of a SHOWROOM livestream downloader. Supports live rooms (RTMP/HLS) and recorded episodes. Spiritual successor to the private/deprecated TypeScript `sba-stream` project, scoped to SHOWROOM only.

## Build & test

```sh
go build -o sba-showroom .
go test ./...
```

## Structure

```
main.go                        Entry point
cmd/
  root.go                      Cobra root command
  download.go                  download sub-command, flag wiring, time parsing
internal/
  showroom/
    api.go                     SHOWROOM REST API types and HTTP helpers
    episode.go                 Recorded episode: HTML scraping + streaming URL
    downloader.go              Orchestration: standby polling → stream selection → ffmpeg loop
  runner/
    ffmpeg.go                  ffmpeg process wrapper
```

## Key design notes

- `internal/showroom/api.go` exposes `cdnBaseURL` and `showroomBaseURL` package-level vars. Tests override these with `httptest.NewServer` URLs — no real HTTP in tests.
- `selectStream` is extracted from the retry loop in `resolveStreamURL` so stream-selection logic (RTMP/HLS priority, quality sorting) is testable as a pure function.
- File naming matches the original `sba-stream`: `YYMMDD-{name}-{4hex}.mp4`.
- For episodes, CloudFront cookies from the streaming URL response are forwarded to ffmpeg via `-headers`.

## APIs used

| Endpoint | Purpose |
|----------|---------|
| `https://public-api.showroom-cdn.com/room/{key}` | Room info + live status |
| `https://www.showroom-live.com/api/live/streaming_url?room_id={id}` | Live stream URLs |
| `https://www.showroom-live.com/episode/watch?id={id}` | Episode page (HTML, scraped for `data-episode`) |
| `https://www.showroom-live.com/api/episode/streaming_url?episode_id={id}` | Episode HLS URL + CloudFront cookies |
