# Changelog

Notable changes to `go-curseforge`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-10-31

### Added

- `Mod.Available` is false only when the API said `isAvailable: false`.

### Fixed

- A mod whose answer omits `isAvailable` no longer reads as unavailable; `Mod.IsAvailable` is true then too.

## [0.1.0] - 2026-10-30

### Added

- `Client` calls the CurseForge API with the key in a header only: `Search` (class, category, game-version and sort
  filters), `Mod`, `Mods`, `Files`, `File`, `Description`, `Changelog`, `Classes`, `Categories` and `DownloadURL`.
- Every answer is capped at 16 MiB, redirects to another origin are refused, and failures are
  `httpkit.StatusError` or `httpkit.RateLimitError`; `Denied` spots a 401 or 403.
- `Mod`, `File`, `Category` and the other response types carry the full API fields; `Mod.Restricted` reports an author
  who forbids downloads outside CurseForge.
- `Newest` picks a mod's newest file by release type and game version; `File.SHA1` and `File.Required` read a file.
- `ParseLink` reads `curseforge://install` links.

[Unreleased]: https://github.com/Rethunk-Tech/go-curseforge/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Rethunk-Tech/go-curseforge/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Rethunk-Tech/go-curseforge/releases/tag/v0.1.0
