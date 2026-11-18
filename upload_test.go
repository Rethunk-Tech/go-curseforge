package curseforge

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-safefs"
)

func writeZip(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "Ab-ShopDrop-1.2.0.zip")
	if err := safefs.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUploadSendsMetadataAndFileAsAForm(t *testing.T) {
	var meta map[string]any
	var file, name, token, agent, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, token, agent = r.URL.Path, r.Header.Get("X-Api-Token"), r.Header.Get("User-Agent")
		form, err := r.MultipartReader()
		if err != nil {
			t.Error(err)
			return
		}
		for {
			part, err := form.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("multipart: %v", err)
				break
			}
			b, _ := io.ReadAll(io.LimitReader(part, 1<<20))
			switch part.FormName() {
			case "metadata":
				_ = json.Unmarshal(b, &meta)
			case "file":
				file, name = string(b), part.FileName()
			}
		}
		_, _ = w.Write([]byte(`{"id":4242}`))
	}))
	defer srv.Close()
	u := NewUploader(UploadConfig{Token: "tok", BaseURL: srv.URL, HTTP: srv.Client(), UserAgent: "ua/1"})
	id, err := u.Upload(t.Context(), writeZip(t, "0123456789"), 31, "ShopDrop 1.2.0", "- fixed it")
	if err != nil || id != 4242 {
		t.Fatalf("id = %d, %v", id, err)
	}
	if path != "/api/projects/31/upload-file" || token != "tok" || agent != "ua/1" || file != "0123456789" || name != "Ab-ShopDrop-1.2.0.zip" {
		t.Errorf("path %q, token %q, agent %q, file %q (%q)", path, token, agent, file, name)
	}
	if fmt.Sprint(meta) != "map[changelog:- fixed it changelogType:markdown displayName:ShopDrop 1.2.0 releaseType:release]" {
		t.Errorf("metadata = %v", meta)
	}
}

func TestUploadRejectsAZeroFileID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":0}`))
	}))
	defer srv.Close()
	id, err := NewUploader(UploadConfig{Token: "t", BaseURL: srv.URL, HTTP: srv.Client()}).Upload(t.Context(), writeZip(t, "x"), 1, "n", "")
	if err == nil || id != 0 {
		t.Fatalf("zero file id: id=%d err=%v", id, err)
	}
}

func TestUploadRefusalsAreTyped(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
		_, err := NewUploader(UploadConfig{Token: "t", BaseURL: srv.URL, HTTP: srv.Client()}).Upload(t.Context(), writeZip(t, "x"), 1, "n", "")
		srv.Close()
		if status == http.StatusTooManyRequests {
			if !errors.Is(err, httpkit.ErrRateLimited) {
				t.Errorf("429: err = %v", err)
			}
		} else if !Denied(err) {
			t.Errorf("%d: err = %v", status, err)
		}
	}
}

func TestUploadNeedsATokenAndAHost(t *testing.T) {
	if _, err := NewUploader(UploadConfig{BaseURL: "https://x"}).Upload(t.Context(), "p", 1, "n", ""); err == nil {
		t.Error("no token accepted")
	}
	if _, err := NewUploader(UploadConfig{Token: "t"}).Upload(t.Context(), "p", 1, "n", ""); err == nil {
		t.Error("no host accepted")
	}
}

func TestAuthorFilesURLPointsAtTheProjectFilesPage(t *testing.T) {
	t.Parallel()
	if got := AuthorFilesURL(31); got != "https://authors.curseforge.com/#/projects/31/files" {
		t.Fatalf("got %q", got)
	}
}

func TestUploadReportsAMissingFile(t *testing.T) {
	t.Parallel()
	_, err := NewUploader(UploadConfig{Token: "t", BaseURL: "https://minecraft.curseforge.com"}).
		Upload(t.Context(), filepath.Join(t.TempDir(), "gone.zip"), 1, "n", "")
	if err == nil {
		t.Fatal("missing zip accepted")
	}
}

func TestUploadTimeoutGrowsWithSize(t *testing.T) {
	t.Parallel()
	if got := uploadTimeout(0); got != 30*time.Second {
		t.Fatalf("empty file: %v", got)
	}
	if got := uploadTimeout(128 << 20); got != 30*time.Second+1024*time.Second {
		t.Fatalf("128 MiB: %v", got)
	}
}
