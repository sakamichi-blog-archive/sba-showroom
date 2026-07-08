# sba-showroom

[SHOWROOM](https://www.showroom-live.com) livestream downloader. Downloads as soon as a stream starts.

## Requirements

- [FFmpeg](https://ffmpeg.org/) — must be on `$PATH`

## Installation

### GitHub Releases

Download a pre-built binary from the [Releases](https://github.com/sakamichi-blog-archive/sba-showroom/releases) page.

### go install

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
```

### EXPECTED_TIME formats

```
HH:mm
YYYY-MM-DD HH:mm
YYYY/MM/DD HH:mm
```

When provided, the downloader waits until the given time before polling aggressively for the stream to go live.

### Flags

Flags must appear before the URL.

| Flag | Default | Description |
|------|---------|-------------|
| `--no-retry` | `false` | Stop after stream ends (retry is on by default) |

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

```

## Output

Files are named `YYMMDD-{room}-{4hex}.mp4` (date in JST).

## License

MIT
