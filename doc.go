// Package curseforge is a client for the CurseForge Core API (v1): mod search, mods, files, categories and
// classes, download addresses, and curseforge:// install links.
//
// The API key rides in the X-Api-Key header only, so no error or log line quotes it. A file whose author forbids
// distribution outside CurseForge has no download address; Mod.Restricted and the ErrNoDownload result of
// Client.DownloadURL report it, and this package never fetches around it. Mod files are never downloaded here.
package curseforge
