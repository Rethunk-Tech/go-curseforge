# HUMANS.md

Run and use `go-curseforge`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-curseforge
```

```go
package main

import (
 "context"
 "errors"
 "fmt"
 "net/http"
 "os"

 "github.com/Rethunk-Tech/go-curseforge"
 "github.com/Rethunk-Tech/go-httpkit"
)

func main() {
 c := curseforge.New(curseforge.Config{
  APIKey:    os.Getenv("CURSEFORGE_API_KEY"),
  HTTP:      &http.Client{Transport: &httpkit.Transport{UserAgent: "MyApp/1.0.0"}},
  UserAgent: "MyApp/1.0.0",
 })
 ctx := context.Background()

 res, err := c.Search(ctx, curseforge.SearchQuery{
  GameID: 669, ClassID: 4643, Text: "content patcher", SortField: curseforge.SortTotalDownloads,
 })
 if err != nil || len(res.Mods) == 0 {
  fmt.Println(res.Pagination.TotalCount, err)
  return
 }
 mod := res.Mods[0]
 file, ok := curseforge.Newest(mod.LatestFiles, nil, curseforge.ReleaseStable)
 if !ok || mod.Restricted() {
  fmt.Println(mod.Name, "must be downloaded from", mod.Links.WebsiteURL)
  return
 }
 link := file.DownloadURL
 if link == "" {
  link, err = c.DownloadURL(ctx, mod.ID, file.ID)
 }
 if errors.Is(err, curseforge.ErrNoDownload) || curseforge.Denied(err) {
  fmt.Println(mod.Name, "must be downloaded from", mod.Links.WebsiteURL)
  return
 }
 fmt.Println(file.FileName, file.SHA1(), link, err)
}
```

## Contracts

| Behaviour | Contract |
| --- | --- |
| `New(Config)` | Trims the key; an empty key makes every call return `ErrNoKey` before any request. A `Client` holds no state, so build a new one when the key changes. |
| API key | Sent as `X-Api-Key` only; never in a URL, an error or a log line. |
| Redirects | Chained onto `Config.HTTP` with `httpkit.SameOrigin`: a redirect to another scheme or host fails. |
| Errors | 429 is `*httpkit.RateLimitError` (reset from `Retry-After`, else zero); any other non-200 is `*httpkit.StatusError`; `Denied` is true for 401 and 403. The body of a refusal is not read. |
| Answers | At most 16 MiB (`safefs.ErrTooLarge` beyond); each request has a 20 s timeout on top of your context. |
| `Search` | `Index+PageSize` past `MaxWindow` is `ErrBeyondWindow` with no request; zero `PageSize` is 50. `CategoryIDs` are ANDed by the API; there is no exclude filter. |
| `Mods` | One POST for all ids; the API caps the batch, so split long lists. Unknown ids are absent. |
| `DownloadURL` | A 403 (see `Denied`) or an empty address means the author forbids third-party downloads; the empty case is `ErrNoDownload`. |
| `Mod.Restricted` | True only when the API said `allowModDistribution: false`; unknown is not restricted. |
| `Mod.Available` | False only when the API said `isAvailable: false`; an omitted field is available. |
| `Newest` | Skips unavailable files and, when `gameVersions` is non-empty, files for other versions. At `ReleaseStable` a stable file beats a newer beta; with none, the newest other file stands in. Higher values take the newest file up to that release type. |
| `ParseLink` | Accepts only `curseforge://install?addonId=<n>&fileId=<n>` with positive ids; any other host, path, userinfo or fragment is refused. |
| `Uploader` | `NewUploader` publishes a zip through the author upload API (`X-Api-Token`, not the API key) and returns the new file's id; `AuthorFilesURL` names the project's files page. |
| Descriptions, changelogs | HTML as the site stores them; this package does not sanitise or convert them. |
| Mod files | Never fetched or written by this package. |

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
