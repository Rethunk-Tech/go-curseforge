<h1 align="center">go-curseforge</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

A client for the CurseForge API: mods, files, categories and download links, with the distribution flag and API-key header handled. Package `curseforge`, module `github.com/Rethunk-Tech/go-curseforge`.

CurseForge sends the API key in a header, forbids downloads for some mods, and answers with large JSON. This client
puts the key in a header only, so no error or log line quotes it, caps every answer at 16 MiB, refuses redirects to
another origin, and reports a file whose author forbids distribution instead of fetching around it. It never downloads
a mod file.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-curseforge
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- Search with class, category, game-version and sort filters, up to the API's 10,000-row window.
- Mods (one or a batch), files, descriptions, changelogs, classes and categories, with the full response fields.
- `Client.DownloadURL` and `Mod.Restricted` tell a distributable file from one the author keeps on CurseForge.
- Typed errors: `httpkit.StatusError` (404 matches `httpkit.ErrNotFound`) and `httpkit.RateLimitError` with `Retry-After`.
- `ParseLink` reads `curseforge://install` links; `Newest` picks a mod's newest file by release channel.
- `Uploader`, built by `NewUploader`, publishes a zip to a project through the author upload API (`X-Api-Token`); `AuthorFilesURL` names the project's files page.

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
