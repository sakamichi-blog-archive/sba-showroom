# CLAUDE.md

## Project

Go implementation of a SHOWROOM livestream downloader. Downloads live HLS streams; supports single-room download and multi-room campaign watching. Spiritual successor to the private/deprecated TypeScript `sba-stream` project, scoped to SHOWROOM only.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for branch naming, commit conventions, and release process.

## Build & test

```sh
go build -o sba-showroom .
mise run test
```

## Structure

```
main.go                        Entry point
cmd/
  root.go                      Command dispatcher (flag.FlagSet, not Cobra)
  download.go                  download sub-command, flag wiring, time parsing
  watch.go                     watch sub-command, --campaign flag, URL args
internal/
  showroom/
    api.go                     SHOWROOM REST API types, HTTP helpers, selectBestHLS
    campaign.go                Campaign slug → rooms.json fetch and parse
    downloader.go              download command: standby polling → HLS resolution → ffmpeg loop
    watcher.go                 watch command: concurrent per-room goroutines, signal handling
  runner/
    ffmpeg.go                  ffmpeg process wrapper (FFmpegProcess, StartFFmpeg, FFmpeg, Kill)
    proc_unix.go               Setpgid: true — isolates ffmpeg from terminal Ctrl+C (non-Windows)
    proc_windows.go            No-op detachProcess (Windows)
```

## Key design notes

- `internal/showroom/api.go` exposes `cdnBaseURL` and `showroomBaseURL` package-level vars. Tests override these with `httptest.NewServer` URLs — no real HTTP in tests.
- `selectBestHLS` in `api.go` picks the highest-quality HLS stream from the streaming URL list. It is the single source of stream-selection logic, shared by both `downloader.go` and `watcher.go`.
- File naming matches the original `sba-stream`: `YYMMDD-{name}-{4hex}.mp4` (date in JST).
- `watch` starts ffmpeg with `Detached: true` so terminal Ctrl+C does not propagate to ffmpeg. `Stop()` sends SIGINT; falls back to `Kill()` if signal delivery fails.
- `waitForLive` in `downloader.go` implements schedule passthrough: when `remaining <= 0` (scheduled time has passed), it returns `ctx.Err()` immediately without polling `is_live`, matching sba-stream's Phase 1→2 transition. Callers proceed straight to stream URL polling.

## APIs used

| Endpoint | Purpose |
|----------|---------|
| `https://public-api.showroom-cdn.com/room/{key}` | Room info + live status |
| `https://www.showroom-live.com/api/live/streaming_url?room_id={id}` | Live HLS stream URLs |
| `https://campaign.showroom-live.com/nogizaka46_sr/data/rooms.json` (nogi) | Campaign room URL keys |
| `https://campaign.showroom-live.com/hinatazaka46_sr/data/rooms.json` (hinata) | |
| `https://campaign.showroom-live.com/sakurazaka46_sr/data/rooms.json` (sakura) | | 
