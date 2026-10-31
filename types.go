package curseforge

import (
	"encoding/json"
	"slices"
	"strings"
)

// Values of ModsSearchSortField (https://docs.curseforge.com/rest-api/#tocS_ModsSearchSortField).
const (
	SortFeatured        = 1
	SortPopularity      = 2
	SortLastUpdated     = 3
	SortName            = 4
	SortAuthor          = 5
	SortTotalDownloads  = 6
	SortCategory        = 7
	SortGameVersion     = 8
	SortEarlyAccess     = 9
	SortFeaturedRelease = 10
	SortReleasedDate    = 11
	SortRating          = 12
)

// Values of FileReleaseType.
const (
	ReleaseStable = 1
	ReleaseBeta   = 2
	ReleaseAlpha  = 3
)

// Values of FileRelationType.
const (
	RelationEmbeddedLibrary = 1
	RelationOptional        = 2
	RelationRequired        = 3
	RelationTool            = 4
	RelationIncompatible    = 5
	RelationInclude         = 6
)

// Values of HashAlgo.
const (
	HashSHA1 = 1
	HashMD5  = 2
)

// Values of ModStatus.
const (
	StatusNew             = 1
	StatusChangesRequired = 2
	StatusUnderSoftReview = 3
	StatusApproved        = 4
	StatusRejected        = 5
	StatusChangesMade     = 6
	StatusInactive        = 7
	StatusAbandoned       = 8
	StatusDeleted         = 9
	StatusUnderReview     = 10
)

// Mod is a CurseForge project. Dates are the API's RFC 3339 strings, which compare correctly as text.
type Mod struct {
	ID                int    `json:"id"`
	GameID            int    `json:"gameId"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	Summary           string `json:"summary"`
	Status            int    `json:"status"`
	DownloadCount     int    `json:"downloadCount"`
	ThumbsUpCount     int    `json:"thumbsUpCount"`
	PrimaryCategoryID int    `json:"primaryCategoryId"`
	ClassID           int    `json:"classId"`
	MainFileID        int    `json:"mainFileId"`
	DateCreated       string `json:"dateCreated"`
	DateModified      string `json:"dateModified"`
	DateReleased      string `json:"dateReleased"`
	// AllowModDistribution is nil when the API did not say; false forbids downloads outside CurseForge.
	AllowModDistribution *bool `json:"allowModDistribution"`
	// IsAvailable is what the API said, or true when it omitted the field. Use Available.
	IsAvailable bool `json:"isAvailable"`
	IsFeatured  bool `json:"isFeatured"`
	available   *bool
	Links       struct {
		WebsiteURL string `json:"websiteUrl"`
		WikiURL    string `json:"wikiUrl"`
		IssuesURL  string `json:"issuesUrl"`
		SourceURL  string `json:"sourceUrl"`
	} `json:"links"`
	Authors []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"authors"`
	Logo        Asset      `json:"logo"`
	Screenshots []Asset    `json:"screenshots"`
	Categories  []Category `json:"categories"`
	LatestFiles []File     `json:"latestFiles"`
}

// Asset is an image of a mod.
type Asset struct {
	ID           int    `json:"id"`
	ModID        int    `json:"modId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ThumbnailURL string `json:"thumbnailUrl"`
	URL          string `json:"url"`
}

// Category is a category, or when IsClass is true a class (a top-level kind of content such as "Mods").
type Category struct {
	ID               int    `json:"id"`
	GameID           int    `json:"gameId"`
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	URL              string `json:"url"`
	IconURL          string `json:"iconUrl"`
	DateModified     string `json:"dateModified"`
	IsClass          bool   `json:"isClass"`
	ClassID          int    `json:"classId"`
	ParentCategoryID int    `json:"parentCategoryId"`
	DisplayIndex     int    `json:"displayIndex"`
}

// Hash is one checksum of a file.
type Hash struct {
	Value string `json:"value"`
	// Algo is HashSHA1 or HashMD5.
	Algo int `json:"algo"`
}

// Dependency is a relation between a file and another mod.
type Dependency struct {
	ModID int `json:"modId"`
	// RelationType is one of the Relation constants.
	RelationType int `json:"relationType"`
}

// File is one published file of a mod.
type File struct {
	ID          int    `json:"id"`
	GameID      int    `json:"gameId"`
	ModID       int    `json:"modId"`
	IsAvailable bool   `json:"isAvailable"`
	DisplayName string `json:"displayName"`
	FileName    string `json:"fileName"`
	// ReleaseType is ReleaseStable, ReleaseBeta or ReleaseAlpha.
	ReleaseType   int    `json:"releaseType"`
	FileStatus    int    `json:"fileStatus"`
	Hashes        []Hash `json:"hashes"`
	FileDate      string `json:"fileDate"`
	FileLength    int64  `json:"fileLength"`
	DownloadCount int64  `json:"downloadCount"`
	// DownloadURL is empty when the author forbids distribution outside CurseForge.
	DownloadURL  string       `json:"downloadUrl"`
	GameVersions []string     `json:"gameVersions"`
	Dependencies []Dependency `json:"dependencies"`
}

// Pagination is the position of a page within a result.
type Pagination struct {
	Index       int `json:"index"`
	PageSize    int `json:"pageSize"`
	ResultCount int `json:"resultCount"`
	TotalCount  int `json:"totalCount"`
}

// UnmarshalJSON reads a Mod and remembers whether the API sent isAvailable.
func (m *Mod) UnmarshalJSON(b []byte) error {
	type plain Mod
	aux := struct {
		*plain
		IsAvailable *bool `json:"isAvailable"`
	}{plain: (*plain)(m)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	m.available = aux.IsAvailable
	m.IsAvailable = aux.IsAvailable == nil || *aux.IsAvailable
	return nil
}

// Available reports that the mod can be used: false only when the API said isAvailable: false. A mod built in code
// without that answer is available.
func (m Mod) Available() bool { return m.available == nil || *m.available }

// Restricted reports that the author forbids downloads outside CurseForge. An unknown flag is not restricted.
func (m Mod) Restricted() bool { return m.AllowModDistribution != nil && !*m.AllowModDistribution }

// CategoryNames lists the names of the mod's categories.
func (m Mod) CategoryNames() []string {
	names := make([]string, len(m.Categories))
	for i, c := range m.Categories {
		names[i] = c.Name
	}
	return names
}

// SHA1 is the file's SHA-1, lower-case hex, or "" when the API lists none.
func (f File) SHA1() string {
	for _, h := range f.Hashes {
		if h.Algo == HashSHA1 && h.Value != "" {
			return strings.ToLower(h.Value)
		}
	}
	return ""
}

// Required lists the ids of the mods this file requires.
func (f File) Required() []int {
	var ids []int
	for _, dep := range f.Dependencies {
		if dep.RelationType == RelationRequired && dep.ModID > 0 {
			ids = append(ids, dep.ModID)
		}
	}
	return ids
}

// Newest picks the newest available file, keeping to files for one of gameVersions when it names any.
// With maxRelease ReleaseStable a stable file wins over a newer beta, and a mod with no stable file yields its
// newest other one; a higher maxRelease takes the newest file whose release type is at most maxRelease.
func Newest(files []File, gameVersions []string, maxRelease int) (File, bool) {
	var best File
	found := false
	for _, f := range files {
		if !f.IsAvailable || f.ID == 0 {
			continue
		}
		if len(gameVersions) > 0 && !slices.ContainsFunc(f.GameVersions, func(v string) bool { return slices.Contains(gameVersions, v) }) {
			continue
		}
		if maxRelease > ReleaseStable {
			if f.ReleaseType <= maxRelease && (!found || f.FileDate > best.FileDate) {
				best, found = f, true
			}
			continue
		}
		stable, bestStable := f.ReleaseType == ReleaseStable, best.ReleaseType == ReleaseStable
		if !found || (stable && !bestStable) || (stable == bestStable && f.FileDate > best.FileDate) {
			best, found = f, true
		}
	}
	return best, found
}
