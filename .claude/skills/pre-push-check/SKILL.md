---
name: pre-push-check
description: Check whether docs and tests are up to date before a git push. Use before pushing a branch.
---

Check whether docs and tests need updating before a git push.

1. Run `git diff @{u}...HEAD --name-only` to list files changed in this push. If there is no upstream yet, fall back to `git diff origin/main...HEAD --name-only`.
2. Per CONTRIBUTING.md: if any files under `cmd/` changed (commands or flags), `README.md` must be updated.
3. Per CONTRIBUTING.md: new code must be accompanied by tests. If non-test files under `internal/` changed without corresponding `_test.go` changes, flag it.
4. If anything looks stale, report what needs updating and block the push. Otherwise confirm everything is in order and allow it.
