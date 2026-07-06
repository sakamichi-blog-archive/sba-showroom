# sba-showroom

SHOWROOM livestream downloader. Downloads as soon as a stream starts, or immediately for recorded episodes.

## Requirements

- [FFmpeg](https://ffmpeg.org/) — must be on `$PATH`

## Installation

### GitHub Releases

Download a pre-built binary from the [Releases](https://github.com/sakamichi-blog-archive/sba-showroom/releases) page.

### go install

Requires Go 1.21+.

```sh
go install github.com/sakamichi-blog-archive/sba-showroom@latest
```

## Usage

```
sba-showroom download [flags] URL [EXPECTED_TIME]
```

### URL formats

```
https://www.showroom-live.com/ROOM_URL_KEY
https://www.showroom-live.com/r/ROOM_URL_KEY
https://www.showroom-live.com/episode/watch?id=ID
```

### EXPECTED_TIME formats

```
HH:mm
YYYY-MM-DD HH:mm
YYYY/MM/DD HH:mm
```

When provided, the downloader waits until the given time before polling aggressively for the stream to go live.

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--hls` | | `false` | Prefer HLS over RTMP |
| `--no-retry` | | `false` | Stop after stream ends (retry is on by default) |

### Examples

```sh
# Download a livestream room
sba-showroom download https://www.showroom-live.com/46_iwamotorenka

# With r/ prefix
sba-showroom download https://www.showroom-live.com/r/46_iwamotorenka

# Wait for a scheduled stream (retries on disconnect by default)
sba-showroom download https://www.showroom-live.com/46_iwamotorenka 17:30

# Download once without retrying
sba-showroom download https://www.showroom-live.com/46_iwamotorenka --no-retry

# Download a recorded episode
sba-showroom download https://www.showroom-live.com/episode/watch?id=14

```

## Output

Files are named `YYMMDD-{room}-{4hex}.mp4`. For episodes, the file modification time is set to the stream's start time.

## License

MIT
