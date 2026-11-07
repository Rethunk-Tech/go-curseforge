# Changelog

Notable changes to `go-curseforge`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-11-07

### Added

- `CheckDownloadURL` and `ErrInsecureURL`.

### Changed

- `DownloadURL` returns `ErrInsecureURL` for an address that is not https or carries credentials. `File.DownloadURL` is still the API's text; pass it to `CheckDownloadURL` and verify the download against `File.Hashes`.
- A `Client` or `Uploader` whose base URL is not https (loopback http excepted) fails every call with `ErrInsecureURL`, so the key or token is never sent in the clear.

### Fixed

- `Search` compared `Index+PageSize` to the window with an addition that could wrap; a huge `Index` now returns `ErrBeyondWindow`.

## [0.3.0] - 2026-11-02

### Added

- `Uploader`, built by `NewUploader`, publishes a zip to a project through the author upload API (`X-Api-Token`, a metadata and file form) and returns the new file's id; `AuthorFilesURL` names the project's files page.

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

[Unreleased]: https://github.com/Rethunk-Tech/go-curseforge/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/Rethunk-Tech/go-curseforge/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/Rethunk-Tech/go-curseforge/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/Rethunk-Tech/go-curseforge/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Rethunk-Tech/go-curseforge/releases/tag/v0.1.0
