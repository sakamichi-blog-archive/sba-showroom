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

## GitHub Actions

When adding or updating GitHub Actions, pin them to a commit SHA using [pinact](https://github.com/suzuki-shunsuke/pinact):

```sh
pinact run .github/workflows/*.yml
```
