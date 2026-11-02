# AGENTS.md

Package `curseforge` is a client for the CurseForge Core API (v1): search, mods, files, categories, download
addresses and `curseforge://install` links. Mod managers and update checkers import it. It imports no consumer package
and reads a file only to upload it (`Uploader`); caches, search merging, scoring and any wording a person reads belong to the caller.

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH.

## Shape

- `Client.do` is the single path for every API call. It refuses with `ErrNoKey` before building a request, puts the
  key in a header only (a transport `*url.Error` quotes the URL, so a key in the query would leak), caps the body with
  `httpkit.DoCapped`, and stops a 429 in the `DoCapped` check so its body is never read.
- Errors are httpkit's types, not package ones: callers already handle `StatusError` and `RateLimitError`. 403 is a
  `StatusError`, not a rate limit, because the download-url endpoint uses it for "author forbids distribution".
- The redirect policy is `httpkit.SameOrigin`, chained onto the caller's client, so `X-Api-Key` stays on the API origin.
- `Mod.AllowModDistribution` is a `*bool` because absent and false differ: only an explicit false is `Restricted`.
- Dates stay RFC 3339 strings: they sort correctly as text and a caller that needs `time.Time` parses what it uses.
- No function returns mod-file bytes; `DownloadURL` returns an address and never follows it.
- Ids are `int` so a path segment is always digits; no caller text reaches a URL path.
- `Newest` takes the release type as the API's own numbers (`ReleaseStable`..`ReleaseAlpha`) so the package carries no
  consumer channel vocabulary.

##| File | Holds |
| --- | --- |
| `doc.go` | Package doc |
| `client.go` | `Client`, `Config`, `do`, and every API call |
| `types.go` | Response types, constants (sort, release, relation, hash, status), `Newest` and the file helpers |
| `link.go` | `ParseLink`, `Ref` |
