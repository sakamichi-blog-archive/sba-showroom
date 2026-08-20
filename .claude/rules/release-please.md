---
paths:
  - release-please-config.json
  - .github/workflows/release-please.yml
  - CONTRIBUTING.md
---

`changelog-sections` controls which commit types appear in the rendered changelog, which has two effects:

1. **Display** — only listed types show up in the changelog body.
2. **Release PR gate** — `src/strategies/base.ts` suppresses the release PR entirely if the rendered changelog is empty (`changelogEmpty` gate). So a commit type not listed here will not trigger a release PR on its own.

The version bump level comes from a separate mechanism in `src/versioning-strategies/default.ts`: only `feat`/breaking changes affect the bump level; everything else falls through to patch. `changelog-sections` has no `bump` property and does not influence the bump level.

**Why `deps` is listed:** Dependabot GitHub Actions commits use the `deps:` prefix (see CONTRIBUTING.md). Without a `changelog-sections` entry, they produce an empty changelog and are silently skipped. Adding `deps` here lets them appear in the changelog and trigger patch release PRs.

**Why `chore`, `ci`, `test`, `refactor`, `docs` are not listed:** Omitting them is intentional. These types have no user-facing impact and should not trigger releases on their own. Commits of these types will not produce a release PR by themselves — they will be bundled into the next release triggered by a listed type. A push containing only these types leaves any existing open release PR **untouched** — `Manifest.createPullRequests()` returns early on an empty candidate list without updating or closing it (`src/manifest.ts`).

## `"draft": true` and the build/publish jobs in release-please.yml

GitHub's immutable releases lock a release's assets the instant it's published — `gh release upload` gets HTTP 422 on an already-published release. Draft releases aren't immutable, and a draft's creation doesn't fire an Actions `release` event at all, so the only workable shape is: release-please creates the release as a **draft** (`"draft": true` here), then `release-please.yml`'s `build` job (gated on `release_created`/`tag_name` outputs) uploads binaries to that still-mutable draft, and a `publish` job flips `--draft=false` only once every platform's binary is attached. (`publish` is not the last job — see the next section for `release-pr`, which runs after it.) Don't move the build/upload logic back to a separate `publish.yml` triggered by `release: published` — that trigger fires after the release is already immutable, which is the original bug.

## Known failure mode: a bogus full-history Release PR, and why `release-pr` is a separate job

`release-please-action`'s `main()` (`src/index.ts`) calls `manifest.createReleases()` and
then, in the same invocation, re-loads the manifest and calls
`manifest.createPullRequests()`. The second call re-queries GitHub for "the latest release"
to know where to stop walking commit history.

With `"draft": true` those two halves are incompatible, for a reason that is verifiable in
release-please's source rather than inferred:

- `releaseGraphQL()` (`src/github-api.ts:563`) ends with `.filter(release => !!release.tagCommit)`.
  A draft release has a null `tagCommit`, so the release the first half just created is
  dropped before `manifest.ts` ever sees it. Note this is **not** a draft filter —
  `isDraft` is selected by the query and mapped onto `ScmRelease.draft`, but it is never
  consumed as a filter anywhere.
- The fallback that should have rescued this fails identically: `backfillReleasesFromTags()`
  (`src/manifest.ts:847`) enumerates real git refs via `GET /repos/{owner}/{repo}/tags`,
  which a draft's non-existent tag also misses.

With both lookups empty, `commitsAfterSha()` (`src/manifest.ts:1813`) hits its fallback bug —
when the last-release SHA isn't found among the walked commits it returns **all** of them
rather than none:

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
bump with the entire project history in its changelog. Because the cause is a code-level
filter rather than API lag, it is deterministic — it fired on both releases cut after
`"draft": true` landed in #39, in each case from the same workflow run that cut the release:

| release | release commit merged (UTC) | published (UTC) | bogus PR (UTC) |
|---------|-----------------------------|-----------------|----------------|
| v0.4.1  | 2026-07-29 13:09:27         | 13:10:52        | #41 at 13:10:07 |
| v0.5.0  | 2026-08-20 03:33:30         | 03:35:11        | #54 at 03:34:21 |

(Times are UTC; this repo's git log renders JST, so `git log` shows these nine hours later.)

**The fix in `release-please.yml`:** the two halves are split across the draft→published
boundary. The `release-please` job passes `skip-github-pull-request: true`, so it only
creates the draft release — `release_created` and `tag_name` are still emitted, since
`outputReleases()` is gated on `skip-github-release`, not on this input. A final `release-pr`
job then runs the action again with `skip-github-release: true`. By then `publish` has
flipped `--draft=false`, the tag exists, `tagCommit` is populated, and the commit walk is
correctly bounded. `createPullRequests()` still runs exactly once per push.

Its gate has three parts, all load-bearing:

```yaml
if: >-
  ${{ !cancelled() && needs.release-please.result == 'success'
  && (needs.publish.result == 'success'
      || needs.release-please.outputs.release_created != 'true') }}
```

- `!cancelled()` — a `skipped` dependency must not skip this job, or no Release PR is ever
  opened on ordinary pushes. Do **not** use the default condition. `always()` also works but
  additionally fires on an explicitly cancelled workflow, which can hit an unpublished draft.
- `release-please.result == 'success'` — no Release PR after a failed release run.
- The `publish`/`release_created` disjunction — a job whose dependency **fails** is reported
  `skipped`, not `failure`, so `publish.result == 'skipped'` means either "no release was
  cut" or "`build` failed". Only the first is safe. Gating on `release_created` separates
  them; without it a failed `build` reproduces the original bug exactly.

Don't collapse these back into one job.

**Residual risk:** the new path assumes GitHub's GraphQL reflects the freshly created tag by
the time `release-pr` queries it. A separate job's runner spin-up is a reasonable buffer, and
unlike the draft window this is a genuine narrow race rather than a certainty — the
"deterministic" above describes the *old* failure and is not a guarantee about the new path.

**If a bogus PR appears anyway:** close it, don't merge it. Confirm it's bogus first with
`git rev-list <last-tag>..main --count` — for #41 this was `0`, i.e. zero real commits since
the release. A `docs:`/`chore:`/etc. commit will not fix or update it (see the
`changelog-sections` section above — empty-candidate pushes leave existing PRs untouched). A
`feat:`/`fix:`/`deps:` commit *might* force a corrected recompute into the same PR, but isn't
guaranteed. Closing it outright is the reliable fix; release-please opens a fresh, correct PR
the next time a triggering commit lands.
