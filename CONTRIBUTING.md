# Contributing

## Branches

Use the `features/NAME` format for branch names.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/) without a scope. Example: `feat: add download command`, not `feat(cmd): add download command`.

## Setup

Install [mise](https://mise.jdx.dev), then:

```sh
mise install
```

## Tasks

mise tasks are the single source of definition for frequently-used scripts, shared between local development and CI. Run `mise tasks` to list them.

## golangci-lint

The golangci-lint version must be kept in sync between `mise.toml` and the `version:` field in `.github/workflows/check.yml`.

## GitHub Actions

GitHub Actions are pinned to commit SHAs using [pinact](https://github.com/suzuki-shunsuke/pinact). The `--min-age 3` flag skips versions released less than 3 days ago.

After adding a new action:

```sh
mise run pinact
```

To update existing actions to their latest versions:

```sh
mise run pinact:update
```
