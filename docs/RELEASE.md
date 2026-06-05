# Release Policy

## Versioning

Spawnling uses **CalVer**: `vYYYY.MM.PATCH`

| Component | Meaning | Example |
|---|---|---|
| `YYYY` | Year | `2026` |
| `MM` | Month (zero-padded) | `06` |
| `PATCH` | Sequential number within the month | `1`, `2`, `3` |

Examples: `v2026.06.1`, `v2026.06.2`, `v2026.07.1`

Increment `PATCH` for every release within the same month (including bugfix/patch releases).
Reset `PATCH` to `1` when the month changes.

## Release Cycle

1. **Feature freeze** — cut a `release/vYYYY.MM.1` branch from `dev`.
2. **Stabilization** — only bugfixes are merged into the release branch. No new features.
3. **Changelog** — update `CHANGELOG.md` with all changes since the last release.
4. **Tagging** — tag the release commit: `git tag vYYYY.MM.PATCH`.
5. **Publish** — push the tag to trigger the release pipeline (GitHub Releases, binaries, Docker image).
6. **Back-merge** — merge any fixes from the release branch back into `dev`.

## Changelog Format

Follow [Keep a Changelog](https://keepachangelog.com) conventions:

```markdown
## [v2026.06.1] - 2026-06-05

### Added
- ...

### Changed
- ...

### Fixed
- ...

### Removed
- ...
```

## Pre-releases

Use suffixes for pre-release versions:

| Suffix | Meaning |
|---|---|
| `-alpha.1` | Early unstable preview |
| `-beta.1` | Feature-complete, testing phase |
| `-rc.1` | Release candidate, code freeze |

Example: `v2026.07.1-beta.1`

Pre-release tags are created on the `release/v*` branch before the final tag.

## Artifacts

Each tagged release should produce:
- GitHub Release with release notes
- Pre-built binaries for Linux, macOS, Windows (amd64, arm64)
- Docker image: `ghcr.io/spawnling/spawnling:v2026.06.1` and `:latest`

## Hotfix Releases

If a critical bug is found after release:

1. Branch from the affected `release/v*`: `hotfix/fix-critical-bug`
2. Fix and test.
3. Merge into the `release/v*` branch.
4. Tag as the next patch: `v2026.06.2`.
5. Back-merge the fix into `dev`.
