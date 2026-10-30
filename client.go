package curseforge

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
)

const (
	// BaseURL is the real API.
	BaseURL = "https://api.curseforge.com/v1"

	// MaxWindow is how deep the API pages: Index plus PageSize may not pass it.
	MaxWindow = 10000
	// MaxPageSize is the most mods one Search request returns.
	MaxPageSize = 50

	requestTimeout = 20 * time.Second
	// maxBody caps one API answer.
	maxBody = 16 << 20
)

var (
	// ErrNoKey means the Client has no API key; nothing is sent.
	ErrNoKey = errors.New("curseforge: no API key")
	// ErrBeyondWindow means a search asked past MaxWindow.
	ErrBeyondWindow = errors.New("curseforge: search window exceeded")
	// ErrNoDownload means the API gave no download address for a file: its author forbids downloads outside
	// CurseForge.
	ErrNoDownload = errors.New("curseforge: file has no download address")
)

// Config is what a Client is built from.
type Config struct {
	// APIKey is sent in the X-Api-Key header. Empty makes every call fail with ErrNoKey.
	APIKey string
	// HTTP sends every request; nil means http.DefaultClient. Redirects to another origin are refused so the key
	// stays with the API.
	HTTP *http.Client
	// BaseURL is the API; empty means BaseURL.
	BaseURL string
	// UserAgent is sent with every request when set.
	UserAgent string
}

// Client calls the CurseForge API. It holds no state beyond its configuration, so it is safe for concurrent use and
// cheap to build again when the key changes.
type Client struct {
	key, base, agent string
	hc               *http.Client
}

// New builds a Client.
func New(cfg Config) *Client {
	return &Client{
		key:   strings.TrimSpace(cfg.APIKey),
		base:  strings.TrimRight(cmp.Or(cfg.BaseURL, BaseURL), "/"),
		agent: cfg.UserAgent,
		hc:    httpkit.WithRedirect(cfg.HTTP, httpkit.SameOrigin),
	}
}

// Denied reports an answer of 401 or 403, which means a bad key or a file the key may not fetch.
func Denied(err error) bool {
	se, ok := errors.AsType[*httpkit.StatusError](err)
	return ok && (se.Code == http.StatusUnauthorized || se.Code == http.StatusForbidden)
}

// do sends one request and decodes the 200 answer into out. A 429 is a *httpkit.RateLimitError and any other
// status a *httpkit.StatusError. The key is in a header only, so no error quotes it.
func (c *Client) do(ctx context.Context, method, path string, params url.Values, body, out any) error {
	if c.key == "" {
		return ErrNoKey
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	u := c.base + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	var payload io.Reader = http.NoBody
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("curseforge: %w", err)
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, payload)
	if err != nil {
		return fmt.Errorf("curseforge: %w", err)
	}
	if c.agent != "" {
		req.Header.Set("User-Agent", c.agent)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Api-Key", c.key)
	raw, err := httpkit.DoCapped(c.hc, req, maxBody, func(resp *http.Response) error {
		if resp.StatusCode != http.StatusTooManyRequests {
			return nil
		}
		e := &httpkit.RateLimitError{}
		if t, ok := httpkit.RetryAfter(resp.Header.Get("Retry-After")); ok {
			e.Reset = t
		}
		return e
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("curseforge: %w", err)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, params, nil, out)
}

// SearchQuery is one /mods/search request. GameID is required; zero fields are not sent.
type SearchQuery struct {
	GameID  int
	ClassID int
	// CategoryIDs keeps mods in every listed category (the API ANDs them; at most 10).
	CategoryIDs []int
	GameVersion string
	Text        string
	// SortField is one of the Sort constants; zero keeps the API's default.
	SortField int
	// Ascending reverses the API's default descending order.
	Ascending bool
	// Index is the offset of the first mod and PageSize the page length, at most MaxPageSize; zero PageSize is 50.
	Index, PageSize int
}

// SearchResult is one page of mods.
type SearchResult struct {
	Mods       []Mod      `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Search lists mods. A ClassID restricts the search to that class.
func (c *Client) Search(ctx context.Context, q SearchQuery) (SearchResult, error) {
	size := cmp.Or(q.PageSize, MaxPageSize)
	if q.Index < 0 || size < 0 || q.Index+size > MaxWindow {
		return SearchResult{}, ErrBeyondWindow
	}
	params := url.Values{}
	params.Set("gameId", strconv.Itoa(q.GameID))
	if q.ClassID > 0 {
		params.Set("classId", strconv.Itoa(q.ClassID))
	}
	if q.Text != "" {
		params.Set("searchFilter", q.Text)
	}
	if q.GameVersion != "" {
		params.Set("gameVersion", q.GameVersion)
	}
	if q.SortField > 0 {
		params.Set("sortField", strconv.Itoa(q.SortField))
		order := "desc"
		if q.Ascending {
			order = "asc"
		}
		params.Set("sortOrder", order)
	}
	params.Set("index", strconv.Itoa(q.Index))
	params.Set("pageSize", strconv.Itoa(size))
	if len(q.CategoryIDs) > 0 {
		raw, err := json.Marshal(q.CategoryIDs)
		if err != nil {
			return SearchResult{}, fmt.Errorf("curseforge: %w", err)
		}
		params.Set("categoryIds", string(raw))
	}
	var out SearchResult
	err := c.get(ctx, "/mods/search", params, &out)
	return out, err
}

// Mod reads one project.
func (c *Client) Mod(ctx context.Context, id int) (Mod, error) {
	var out struct {
		Data Mod `json:"data"`
	}
	err := c.get(ctx, "/mods/"+strconv.Itoa(id), nil, &out)
	return out.Data, err
}

// Mods reads many projects with their latest files in one request (POST /mods). The API caps the batch; split long
// lists. A mod the API does not know is absent from the answer.
func (c *Client) Mods(ctx context.Context, ids []int) ([]Mod, error) {
	var out struct {
		Data []Mod `json:"data"`
	}
	err := c.do(ctx, http.MethodPost, "/mods", nil, struct {
		ModIDs []int `json:"modIds"`
	}{ids}, &out)
	return out.Data, err
}

// Description reads a project's long description, which is HTML.
func (c *Client) Description(ctx context.Context, id int) (string, error) {
	return c.text(ctx, "/mods/"+strconv.Itoa(id)+"/description")
}

// FilesQuery narrows Files. Zero fields are not sent.
type FilesQuery struct {
	GameVersion     string
	Index, PageSize int
}

// FilesResult is one page of a mod's files.
type FilesResult struct {
	Files      []File     `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Files lists a project's files. A zero PageSize is 50, the API's maximum.
func (c *Client) Files(ctx context.Context, modID int, q FilesQuery) (FilesResult, error) {
	params := url.Values{"pageSize": {strconv.Itoa(cmp.Or(q.PageSize, MaxPageSize))}}
	if q.GameVersion != "" {
		params.Set("gameVersion", q.GameVersion)
	}
	if q.Index > 0 {
		params.Set("index", strconv.Itoa(q.Index))
	}
	var out FilesResult
	err := c.get(ctx, "/mods/"+strconv.Itoa(modID)+"/files", params, &out)
	return out, err
}

// File reads one file of a project.
func (c *Client) File(ctx context.Context, modID, fileID int) (File, error) {
	var out struct {
		Data File `json:"data"`
	}
	err := c.get(ctx, fileBase(modID, fileID), nil, &out)
	return out.Data, err
}

// Changelog reads one file's changelog, which is HTML.
func (c *Client) Changelog(ctx context.Context, modID, fileID int) (string, error) {
	return c.text(ctx, fileBase(modID, fileID)+"/changelog")
}

// DownloadURL asks the API for a file's address. A refusal (see Denied) or an empty answer means the author forbids
// downloads outside CurseForge; the empty answer is ErrNoDownload.
func (c *Client) DownloadURL(ctx context.Context, modID, fileID int) (string, error) {
	link, err := c.text(ctx, fileBase(modID, fileID)+"/download-url")
	if err == nil && link == "" {
		return "", ErrNoDownload
	}
	return link, err
}

// Classes lists a game's classes (top-level kinds of content, such as "Mods").
func (c *Client) Classes(ctx context.Context, gameID int) ([]Category, error) {
	return c.categories(ctx, url.Values{"gameId": {strconv.Itoa(gameID)}, "classesOnly": {"true"}})
}

// Categories lists the categories of one class of a game.
func (c *Client) Categories(ctx context.Context, gameID, classID int) ([]Category, error) {
	return c.categories(ctx, url.Values{"gameId": {strconv.Itoa(gameID)}, "classId": {strconv.Itoa(classID)}})
}

func (c *Client) categories(ctx context.Context, params url.Values) ([]Category, error) {
	var out struct {
		Data []Category `json:"data"`
	}
	err := c.get(ctx, "/categories", params, &out)
	return out.Data, err
}

func (c *Client) text(ctx context.Context, path string) (string, error) {
	var out struct {
		Data string `json:"data"`
	}
	err := c.get(ctx, path, nil, &out)
	return out.Data, err
}

func fileBase(modID, fileID int) string {
	return "/mods/" + strconv.Itoa(modID) + "/files/" + strconv.Itoa(fileID)
}
