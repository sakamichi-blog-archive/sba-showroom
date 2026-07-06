# Contributing

## Setup

Install [mise](https://mise.jdx.dev), then:

```sh
mise install
```

## Tasks

| Command | Description |
|---------|-------------|
| `mise run build` | Build binary |
| `mise run fmt` | Format Go source files |
| `mise run lint` | Lint Go source files |
| `mise run test` | Run tests |

## golangci-lint

The golangci-lint version must be kept in sync between `mise.toml` and the `version:` field in `.github/workflows/check.yml`.

## GitHub Actions

GitHub Actions are pinned to commit SHAs using [pinact](https://github.com/suzuki-shunsuke/pinact). The `--min-age 3` flag skips versions released less than 3 days ago.

After adding a new action:

```sh
pinact run --min-age 3 .github/workflows/*.yml
```

To update existing actions to their latest versions:

```sh
pinact run --update --min-age 3 .github/workflows/*.yml
```
