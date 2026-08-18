# Contributing

## Branches

Never commit directly to `main`. Always work on a branch and open a pull request.

Use the `features/NAME` format for branch names.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/) without a scope. Example: `feat: add download command`, not `feat(cmd): add download command`.

Use `deps` as the commit type for dependency updates (e.g. `deps: bump golang.org/x/net`).

## Documentation

README usage docs must be kept up to date when commands or flags change.

## Tests

New code must be accompanied by tests. Run the test suite with:

```sh
mise run test
```


## Setup

Install [mise](https://mise.jdx.dev), then:

```sh
mise install
```

## Tasks

mise tasks are the single source of definition for frequently-used scripts, shared between local development and CI. Run `mise tasks` to list them.

## golangci-lint

The golangci-lint version must be kept in sync between `mise.toml` and the `version:` field in `.github/workflows/check.yml`.

## Releasing

Releases are automated via [Release Please](https://github.com/googleapis/release-please):

1. Conventional commits merged to `main` are analyzed by Release Please, which opens or updates a Release PR with a bumped version and updated changelog.
2. Merging the Release PR creates a GitHub Release and tag.
3. The publish workflow triggers on the release and builds binaries for Linux, macOS, and Windows, uploading them to the release.

No manual steps are required — commit messages drive the version bump (feat → minor, fix → patch, breaking change → major).

## GitHub Actions

GitHub Actions are pinned to commit SHAs using [pinact](https://github.com/suzuki-shunsuke/pinact). The `--min-age 3` flag skips versions released less than 3 days ago.

After adding a new action:

```sh
mise run pin:actions
```

To update existing actions to their latest versions:

```sh
mise run pin:actions:update
```
