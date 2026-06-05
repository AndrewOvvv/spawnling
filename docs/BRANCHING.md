# Branching Strategy

Spawnling uses a **Trunk-based development** model with dedicated release branches.

## Branch Structure

```
dev                  ← trunk, continuous integration
feature/<name>       ← short-lived feature branches
release/v2026.06.1   ← stabilization + bugfix only
release/v2026.07.1   ← next release
```

## Branches

### `dev`
- Main trunk branch. All active development is integrated here.
- Direct commits are allowed for trivial changes.
- Default branch for all PRs and feature merges.
- Should always be in a buildable, testable state.

### `feature/<name>`
- Branched from `dev`.
- Short-lived: merged back to `dev` via PR as soon as the feature is ready.
- Naming: `feature/server-manager`, `feature/web-ui`, `feature/auth`, etc.
- Delete after merge.

### `release/v<version>`
- Branched from `dev` when a release cycle begins.
- **Only bugfixes are allowed** — no new features.
- Once stable, the release is tagged: `v2026.06.1`.
- Patch releases increment the patch number: `v2026.06.2`, `v2026.06.3`.
- Branch is kept after tagging (for future patch releases if needed).

### `hotfix/<name>` *(optional)*
- For critical fixes that can't wait for the next release cycle.
- Branched from the relevant `release/v*` branch.
- Merged back into both the release branch and `dev`.

## Rules

| Action | Where |
|---|---|
| New feature development | `feature/*` → `dev` |
| Release stabilization | `release/v*` (bugfix only) |
| Emergency patch | `hotfix/*` → `release/v*` + `dev` |
| Tags | Only on `release/v*` branches |
| Force push | Never on `dev` or `release/*` |

## Example Workflow

```bash
# Start a new feature
git checkout dev
git checkout -b feature/server-api

# ... work, commit ...

# Merge back to dev (via PR)
git checkout dev
git merge --no-ff feature/server-api

# Start a release
git checkout -b release/v2026.06.1
# stabilize, fix bugs only...
git tag v2026.06.1
```
