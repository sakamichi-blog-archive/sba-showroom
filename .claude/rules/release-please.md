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

## Why the Release PR is opened by a separate job

`release-please-action`'s `main()` calls `manifest.createReleases()` (tags + creates the
release for a just-merged Release PR) and then, in the same invocation, re-loads the
manifest and calls `manifest.createPullRequests()` to check whether a new Release PR is
needed. That second call re-queries GitHub for "the latest release" to know where to stop
walking commit history.

With `"draft": true` those two halves are incompatible. **GitHub does not create the git
tag for a draft release until it is published**, so at the moment `createPullRequests()`
runs, the release this same run just created is invisible to the lookup. `commitsAfterSha()`
(`src/manifest.ts`) then hits its fallback bug: when the expected last-release SHA isn't
found among the walked commits, it returns **all** of them rather than none.

```ts
function commitsAfterSha(commits: Commit[], lastReleaseSha: string) {
  const index = commits.findIndex(commit => commit.sha === lastReleaseSha);
  if (index === -1) {
    return commits; // bug: should probably be []
  }
  return commits.slice(0, index);
}
```

The symptom is a second PR opened moments after a real release, proposing a much-too-large
bump with the entire project history dumped into the changelog. This is deterministic, not a
transient API lag — it fired on both releases cut after `"draft": true` landed:

| release | release commit merged | draft window | bogus PR |
|---------|-----------------------|--------------|----------|
| v0.4.1  | 13:09:27              | → 13:10:52   | #41 at 13:10:07 |
| v0.5.0  | 03:33:30              | → 03:35:11   | #54 at 03:34:21 |

Both were created *by the same workflow run* that cut the release, inside the window where
the release was still a draft and the tag did not yet exist.

**The fix in `release-please.yml`:** the two halves are split across the draft→published
boundary. The `release-please` job passes `skip-github-pull-request: true`, so it only tags
and creates the draft release. A final `release-pr` job — `needs: [release-please, publish]`,
gated on `publish` having either succeeded or been skipped — runs the action again with
`skip-github-release: true`. By then `publish` has flipped `--draft=false` and the tag is
real, so the commit walk is correctly bounded. On an ordinary push `build`/`publish` skip and
`release-pr` runs immediately, which is the pre-split behavior.

Don't collapse these back into one job, and don't drop the `always()` from `release-pr`'s
condition — without it the job is skipped whenever `publish` is skipped, i.e. on every
non-release push, and no Release PR is ever opened.

If a bogus PR does appear anyway: close it, don't merge it. A `docs:`/`chore:`/etc. commit
will not fix or update it (see the `changelog-sections` section above — empty-candidate
pushes leave existing PRs untouched). Closing it outright is the reliable fix; release-please
opens a fresh, correct PR the next time a triggering commit lands.
