# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

- Response bodies from the API (every `Client` method): capped at 16 MiB, and no body may panic a decoder (the
  `FuzzDecoders` target). Fields such as `DownloadURL`, `Links` and `Logo` are remote text; this package does not open
  them.
- Redirects: one to another scheme or host is refused, so the API key never leaves the API's origin.
- The API key: sent in the `X-Api-Key` header only. No error this package builds contains it, and the URL it puts in
  transport errors carries only the query, which never holds the key.
- `ParseLink`: pasted or remote text; only the exact `curseforge://install` form with positive ids is accepted
  (`FuzzParseLink`).

## What is out of scope

- The caller supplies and stores the API key, chooses `Config.BaseURL`, and decides what to do with a returned
  `DownloadURL`: validate the scheme and host before fetching, and cap and hash what it downloads.
- Descriptions and changelogs are HTML from the site: sanitise before rendering.
- CurseForge enforces distribution rights; a `Restricted` mod is reported, not enforced.
