package curseforge

import (
	"bytes"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-safefs"
)

const (
	modOpen = `{"data":{"id":10,"gameId":669,"name":"Open Mod","allowModDistribution":true,"links":{"websiteUrl":"https://www.curseforge.com/stardewvalley/mods/open"}}}`
	modShut = `{"data":{"id":20,"name":"Shut Mod","allowModDistribution":false}}`
	file2   = `{"id":102,"modId":10,"isAvailable":true,"displayName":"Open 2.0.zip","fileName":"open-2.0.zip","releaseType":1,"fileDate":"2026-02-01T00:00:00Z","fileLength":2048,"downloadUrl":"https://edge/open-2.0.zip","hashes":[{"value":"AB","algo":1},{"value":"cd","algo":2}],"dependencies":[{"modId":11,"relationType":3},{"modId":12,"relationType":2}]}`
)

func fake(t *testing.T) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "k" {
			t.Errorf("key header %q", r.Header.Get("X-Api-Key"))
		}
		if r.Header.Get("User-Agent") != "test/1" {
			t.Errorf("user agent %q", r.Header.Get("User-Agent"))
		}
		q := r.URL.Query()
		switch r.URL.Path {
		case "/categories":
			if q.Get("classesOnly") == "true" {
				_, _ = w.Write([]byte(`{"data":[{"id":4643,"name":"Mods","isClass":true}]}`))
				return
			}
			if q.Get("gameId") != "669" || q.Get("classId") != "4643" {
				t.Errorf("categories params %v", q)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":6412,"name":"Items","classId":4643}]}`))
		case "/mods/search":
			if q.Get("gameId") != "669" || q.Get("classId") != "4643" || q.Get("searchFilter") != "cp" ||
				q.Get("sortField") != "6" || q.Get("sortOrder") != "asc" || q.Get("index") != "20" || q.Get("pageSize") != "20" ||
				q.Get("categoryIds") != "[6412,6420]" || q.Get("gameVersion") != "1.6" {
				t.Errorf("search params %v", q)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":10,"name":"Open Mod","latestFiles":[` + file2 + `]}],"pagination":{"index":20,"pageSize":20,"resultCount":1,"totalCount":41}}`))
		case "/mods/10":
			_, _ = w.Write([]byte(modOpen))
		case "/mods/20":
			_, _ = w.Write([]byte(modShut))
		case "/mods/10/description":
			_, _ = w.Write([]byte(`{"data":"<p>About</p>"}`))
		case "/mods/10/files":
			if q.Get("pageSize") != "50" {
				t.Errorf("files params %v", q)
			}
			_, _ = w.Write([]byte(`{"data":[` + file2 + `],"pagination":{"totalCount":1}}`))
		case "/mods/10/files/102":
			_, _ = w.Write([]byte(`{"data":` + file2 + `}`))
		case "/mods/10/files/102/changelog":
			_, _ = w.Write([]byte(`{"data":"<p>Fixed</p>"}`))
		case "/mods/10/files/102/download-url":
			_, _ = w.Write([]byte(`{"data":"https://edge/open-2.0.zip"}`))
		case "/mods/10/files/101/download-url":
			http.Error(w, "no", http.StatusForbidden)
		case "/mods/10/files/103/download-url":
			_, _ = w.Write([]byte(`{"data":null}`))
		case "/mods":
			if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("batch request %s %s", r.Method, r.Header.Get("Content-Type"))
			}
			_, _ = w.Write([]byte(`{"data":[{"id":10,"latestFiles":[` + file2 + `]}]}`))
		case "/limited":
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return New(Config{APIKey: " k ", BaseURL: srv.URL + "/", UserAgent: "test/1"})
}

func TestSearchSendsFiltersAndReadsPagination(t *testing.T) {
	t.Parallel()
	res, err := fake(t).Search(t.Context(), SearchQuery{
		GameID: 669, ClassID: 4643, CategoryIDs: []int{6412, 6420}, GameVersion: "1.6", Text: "cp",
		SortField: SortTotalDownloads, Ascending: true, Index: 20, PageSize: 20,
	})
	if err != nil || res.Pagination.TotalCount != 41 || len(res.Mods) != 1 || len(res.Mods[0].LatestFiles) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := fake(t).Search(t.Context(), SearchQuery{GameID: 1, Index: MaxWindow - 10}); !errors.Is(err, ErrBeyondWindow) {
		t.Fatalf("past the window: %v", err)
	}
}

func TestReads(t *testing.T) {
	t.Parallel()
	c, ctx := fake(t), t.Context()
	m, err := c.Mod(ctx, 10)
	if err != nil || m.Name != "Open Mod" || m.GameID != 669 || m.Restricted() {
		t.Fatalf("%+v %v", m, err)
	}
	if m, err = c.Mod(ctx, 20); err != nil || !m.Restricted() {
		t.Fatalf("restricted: %+v %v", m, err)
	}
	if d, err := c.Description(ctx, 10); err != nil || d != "<p>About</p>" {
		t.Fatalf("%q %v", d, err)
	}
	files, err := c.Files(ctx, 10, FilesQuery{})
	if err != nil || len(files.Files) != 1 || files.Files[0].SHA1() != "ab" || files.Files[0].FileLength != 2048 {
		t.Fatalf("%+v %v", files, err)
	}
	if _, err := c.Files(ctx, 10, FilesQuery{GameVersion: "1.6", Index: 50}); err != nil {
		t.Fatal(err)
	}
	if got := (Mod{Categories: []Category{{Name: "Items"}, {Name: "Maps"}}}).CategoryNames(); !slices.Equal(got, []string{"Items", "Maps"}) {
		t.Fatalf("CategoryNames = %v", got)
	}
	if f, err := c.File(ctx, 10, 102); err != nil || f.DisplayName != "Open 2.0.zip" {
		t.Fatalf("%+v %v", f, err)
	}
	if ch, err := c.Changelog(ctx, 10, 102); err != nil || ch != "<p>Fixed</p>" {
		t.Fatalf("%q %v", ch, err)
	}
	mods, err := c.Mods(ctx, []int{10})
	if err != nil || len(mods) != 1 || len(mods[0].LatestFiles) != 1 {
		t.Fatalf("%+v %v", mods, err)
	}
	classes, err := c.Classes(ctx, 669)
	if err != nil || len(classes) != 1 || !classes[0].IsClass {
		t.Fatalf("%+v %v", classes, err)
	}
	cats, err := c.Categories(ctx, 669, 4643)
	if err != nil || len(cats) != 1 || cats[0].Name != "Items" {
		t.Fatalf("%+v %v", cats, err)
	}
}

func TestDownloadURL(t *testing.T) {
	t.Parallel()
	c, ctx := fake(t), t.Context()
	if u, err := c.DownloadURL(ctx, 10, 102); err != nil || u != "https://edge/open-2.0.zip" {
		t.Fatalf("%q %v", u, err)
	}
	if _, err := c.DownloadURL(ctx, 10, 101); !Denied(err) {
		t.Fatalf("a 403 is a refusal: %v", err)
	}
	if _, err := c.DownloadURL(ctx, 10, 103); !errors.Is(err, ErrNoDownload) {
		t.Fatalf("a null address: %v", err)
	}
}

func TestErrors(t *testing.T) {
	t.Parallel()
	c := fake(t)
	if _, err := c.Mod(t.Context(), 99); !errors.Is(err, httpkit.ErrNotFound) || Denied(err) {
		t.Fatalf("404: %v", err)
	}
	rl := &httpkit.RateLimitError{}
	err := c.get(t.Context(), "/limited", nil, &struct{}{})
	if !errors.As(err, &rl) || rl.Reset.Before(time.Now().Add(time.Minute)) {
		t.Fatalf("429: %v", err)
	}
	if _, err := New(Config{}).Mod(t.Context(), 1); !errors.Is(err, ErrNoKey) {
		t.Fatalf("no key: %v", err)
	}
}

func TestKeyStaysOffErrorsAndOtherOrigins(t *testing.T) {
	t.Parallel()
	other := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("the redirect reached another origin with key %q", r.Header.Get("X-Api-Key"))
	}))
	t.Cleanup(other.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	_, err := New(Config{APIKey: "secret-key", BaseURL: srv.URL}).Mod(t.Context(), 1)
	if err == nil || strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("redirect must fail without the key in the error: %v", err)
	}
}

func TestOversizedAnswerIsRefused(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":"`))
		_, _ = w.Write(bytes.Repeat([]byte("a"), maxBody))
		_, _ = w.Write([]byte(`"}`))
	}))
	t.Cleanup(srv.Close)
	if _, err := New(Config{APIKey: "k", BaseURL: srv.URL}).Description(t.Context(), 1); !errors.Is(err, safefs.ErrTooLarge) {
		t.Fatalf("an answer over the cap: %v", err)
	}
}

func TestSearchWindowGuardDoesNotOverflow(t *testing.T) {
	_, err := fake(t).Search(t.Context(), SearchQuery{GameID: 1, Index: math.MaxInt - 10})
	if !errors.Is(err, ErrBeyondWindow) {
		t.Fatalf("overflowing index: %v", err)
	}
}

func TestFilesRejectsANegativeIndex(t *testing.T) {
	if _, err := fake(t).Files(t.Context(), 10, FilesQuery{Index: -1}); !errors.Is(err, ErrBeyondWindow) {
		t.Fatalf("negative index: %v", err)
	}
}

func TestModsEmptyListMakesNoCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Error("POST /mods with no ids")
	}))
	t.Cleanup(srv.Close)
	mods, err := New(Config{APIKey: "k", BaseURL: srv.URL}).Mods(t.Context(), nil)
	if err != nil || mods != nil {
		t.Fatalf("mods=%v err=%v", mods, err)
	}
}

func TestCheckDownloadURLRequiresHTTPSWithoutCredentials(t *testing.T) {
	for _, bad := range []string{"http://edge.example/a.zip", "ftp://x/a", "https://u:p@edge.example/a.zip", "https:///a", "https://:443/a.zip", "//edge/a", "a.zip", "https://%zz"} {
		if _, err := CheckDownloadURL(bad); !errors.Is(err, ErrInsecureURL) {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if got, err := CheckDownloadURL("https://edge.example/a.zip"); err != nil || got != "https://edge.example/a.zip" {
		t.Fatalf("https refused: %q, %v", got, err)
	}
}

func TestPlainHTTPBaseIsRefusedExceptOnLoopback(t *testing.T) {
	for _, base := range []string{"http://api.example/v1", "ftp://api.example", "https://u:p@api.example", "https://:443"} {
		if _, err := New(Config{APIKey: "k", BaseURL: base}).Classes(t.Context(), 1); !errors.Is(err, ErrInsecureURL) {
			t.Errorf("%q: %v", base, err)
		}
		if _, err := NewUploader(UploadConfig{Token: "t", BaseURL: base}).Upload(t.Context(), "x.zip", 1, "n", "c"); !errors.Is(err, ErrInsecureURL) {
			t.Errorf("upload %q: %v", base, err)
		}
	}
	for _, base := range []string{"http://127.0.0.1:1", "http://localhost:1", "http://[::1]:1"} {
		if err := checkBase(base); err != nil {
			t.Errorf("%q: %v", base, err)
		}
	}
}

// The API fills the address in, and the caller fetches it: a plain-http or credentialed address must not come back
// as a download link, even when the API itself is reached over loopback http.
func TestDownloadURLRefusesWhatIsNotAnHTTPSAddress(t *testing.T) {
	t.Parallel()
	answers := map[string]string{
		"1": "http://edge.example/a.zip", "2": "ftp://edge.example/a.zip", "3": "https://" + "u" + ":" + "p" + "@edge.example/a.zip",
		"4": "https:///a.zip", "5": "https://edge.example/a.zip",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/mods/10/files/"), "/download-url")
		_, _ = w.Write([]byte(`{"data":"` + answers[id] + `"}`))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{APIKey: "k", BaseURL: srv.URL})
	for id, link := range answers {
		fileID, _ := strconv.Atoi(id)
		got, err := c.DownloadURL(t.Context(), 10, fileID)
		if id == "5" {
			if err != nil || got != link {
				t.Errorf("https address: %q, %v", got, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("%q came back as a download link", link)
		}
	}
}
