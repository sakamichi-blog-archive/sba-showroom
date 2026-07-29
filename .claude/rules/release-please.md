---
paths:
  - release-please-config.json
---

`changelog-sections` controls which commit types appear in the rendered changelog, which has two effects:

1. **Display** — only listed types show up in the changelog body.
2. **Release PR gate** — `src/strategies/base.ts` suppresses the release PR entirely if the rendered changelog is empty (`changelogEmpty` gate). So a commit type not listed here will not trigger a release PR on its own.

The version bump level comes from a separate mechanism in `src/versioning-strategies/default.ts`: only `feat`/breaking changes affect the bump level; everything else falls through to patch. `changelog-sections` has no `bump` property and does not influence the bump level.

**Why `deps` is listed:** Dependabot GitHub Actions commits use the `deps:` prefix (see CONTRIBUTING.md). Without a `changelog-sections` entry, they produce an empty changelog and are silently skipped. Adding `deps` here lets them appear in the changelog and trigger patch release PRs.

**Why `chore`, `ci`, `test`, `refactor`, `docs` are not listed:** Omitting them is intentional. These types have no user-facing impact and should not trigger releases on their own. Commits of these types will not produce a release PR by themselves — they will be bundled into the next release triggered by a listed type.
