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

### mise

[mise](https://mise.jdx.dev)'s [GitHub backend](https://mise.jdx.dev/dev-tools/backends/github.html) installs the pre-built binary directly from Releases:

```sh
mise use -g github:sakamichi-blog-archive/sba-showroom
```

If `sba-showroom` is too long to type, give it a shorter alias. Add to `mise.toml` (or `~/.config/mise/config.toml` for a global alias):

```toml
[tool_alias]
sba = "github:sakamichi-blog-archive/sba-showroom"
```

Then install under the alias, renaming the binary to match:

```sh
mise use -g "sba[bin=sba]@latest"
```

## Usage

### `download`

Download a single room, waiting for it to go live if it isn't already.

```
sba-showroom download [flags] URL [EXPECTED_TIME]
```

**URL formats**

```
https://www.showroom-live.com/ROOM_URL_KEY
https://www.showroom-live.com/r/ROOM_URL_KEY
```

**EXPECTED_TIME formats**

```
HH:mm
YYYY-MM-DD HH:mm
YYYY/MM/DD HH:mm
```

When provided, the downloader polls every 20 s until the scheduled time, then switches to polling for stream URLs directly.

**Flags** (must appear before the URL)

| Flag | Default | Description |
|------|---------|-------------|
| `--no-retry` | `false` | Stop after stream ends (retry is on by default) |

**Examples**

```sh
# Download a room (retries on disconnect by default)
sba-showroom download https://www.showroom-live.com/46_iwamotorenka

# Wait for a scheduled stream
sba-showroom download https://www.showroom-live.com/46_iwamotorenka 17:30

# Download once without retrying
sba-showroom download --no-retry https://www.showroom-live.com/46_iwamotorenka
```

### `watch` (experimental)

> **Experimental:** `watch` is under active development. Concurrent multi-room recording has had several duplicate-download bugs (see [CHANGELOG](CHANGELOG.md)); for a single critical room, `download` is the more battle-tested choice.

Monitor one or more campaign groups and/or individual rooms, automatically downloading any that go live. Multiple rooms are downloaded concurrently.

```
sba-showroom watch [flags] [URL ...]
```

At least one `--campaign` or room URL is required.

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--campaign` | — | Campaign to watch (`nogi`, `hinata`, `sakura`); may be repeated |
| `--verbose` | `false` | Log room names on startup |

**Examples**

```sh
# Watch all Nogizaka46 rooms
sba-showroom watch --campaign nogi

# Watch multiple campaigns
sba-showroom watch --campaign nogi --campaign hinata

# Watch a campaign plus specific rooms not in it
sba-showroom watch --campaign nogi https://www.showroom-live.com/r/someroom

# Watch specific rooms only
sba-showroom watch https://www.showroom-live.com/46_iwamotorenka https://www.showroom-live.com/46_shibatayuna
```

**Shutdown**

- Ctrl+C with no active downloads exits immediately.
- Ctrl+C with active downloads prints a warning and keeps recording.
- A second Ctrl+C within 3 seconds stops all downloads gracefully and exits.

## Output

Files are named `YYMMDD-{room}-{4hex}.mp4` (date in JST).

## License

MIT
