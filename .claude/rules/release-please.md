---
paths:
  - release-please-config.json
  - .github/workflows/release-please.yml
---

`changelog-sections` controls which commit types appear in the rendered changelog, which has two effects:

1. **Display** — only listed types show up in the changelog body.
2. **Release PR gate** — `src/strategies/base.ts` suppresses the release PR entirely if the rendered changelog is empty (`changelogEmpty` gate). So a commit type not listed here will not trigger a release PR on its own.

The version bump level comes from a separate mechanism in `src/versioning-strategies/default.ts`: only `feat`/breaking changes affect the bump level; everything else falls through to patch. `changelog-sections` has no `bump` property and does not influence the bump level.

**Why `deps` is listed:** Dependabot GitHub Actions commits use the `deps:` prefix (see CONTRIBUTING.md). Without a `changelog-sections` entry, they produce an empty changelog and are silently skipped. Adding `deps` here lets them appear in the changelog and trigger patch release PRs.

**Why `chore`, `ci`, `test`, `refactor`, `docs` are not listed:** Omitting them is intentional. These types have no user-facing impact and should not trigger releases on their own. Commits of these types will not produce a release PR by themselves — they will be bundled into the next release triggered by a listed type. A push containing only these types leaves any existing open release PR **untouched** — `Manifest.createPullRequests()` returns early on an empty candidate list without updating or closing it (`src/manifest.ts`).

## `"draft": true` and the build/publish jobs in release-please.yml

GitHub's immutable releases lock a release's assets the instant it's published — `gh release upload` gets HTTP 422 on an already-published release. Draft releases aren't immutable, and a draft's creation doesn't fire an Actions `release` event at all, so the only workable shape is: release-please creates the release as a **draft** (`"draft": true` here), then `release-please.yml`'s `build` job (gated on `release_created`/`tag_name` outputs) uploads binaries to that still-mutable draft, and a final `publish` job flips `--draft=false` only once every platform's binary is attached. Don't move the build/upload logic back to a separate `publish.yml` triggered by `release: published` — that trigger fires after the release is already immutable, which is the original bug.

## Known failure mode: a release-please run can propose a bogus full-history bump

`release-please-action`'s `main()` calls `manifest.createReleases()` (tags + creates the release for a just-merged Release PR) and then, in the same invocation, re-loads the manifest and calls `manifest.createPullRequests()` to check whether a new Release PR is needed. That second call re-queries GitHub for "the latest release" to know where to stop walking commit history. If that lookup misses the release/tag this same run just created (e.g. GitHub API read-after-write lag), `commitsAfterSha()` (`src/manifest.ts`) has a fallback bug: when the expected last-release SHA isn't found in the walked commits, it returns **all** of them instead of none:

```ts
function commitsAfterSha(commits: Commit[], lastReleaseSha: string) {
  const index = commits.findIndex(commit => commit.sha === lastReleaseSha);
  if (index === -1) {
    return commits; // bug: should probably be []
  }
  return commits.slice(0, index);
}
```

Symptom: right after a real release is cut, release-please opens a second PR proposing a much-too-large version bump (e.g. jumping a full minor version) with the entire project history dumped into the changelog. This happened after cutting v0.4.1 (see PR #41) with zero real commits in between (`git rev-list v0.4.1..main --count` was `0`).

**If you see this: close the bogus PR, don't merge it.** A `docs:`/`chore:`/etc. commit will NOT fix or update it (previous section — empty-candidate pushes don't touch existing PRs). A `feat:`/`fix:`/`deps:` commit might force a corrected recompute into the same PR, but isn't guaranteed. Closing it outright is the reliable fix; release-please will open a fresh, correct PR next time a real triggering commit lands.
