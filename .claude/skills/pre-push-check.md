---
name: pre-push-check
description: Check whether docs need updating before a git push. Use before pushing a branch.
version: 0.1.0
---

Check whether docs need updating before a git push.

1. Run `git diff @{u}...HEAD --name-only` to list files changed in this push. If there is no upstream yet, fall back to `git diff origin/main...HEAD --name-only`.
2. Per CONTRIBUTING.md: if any files under `cmd/` changed (commands or flags), `README.md` must be updated.
3. If README looks stale, report what needs updating and block the push. Otherwise confirm docs are in order and allow it.
