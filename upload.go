package curseforge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-safefs"
)

// maxUploadReply caps the answer to an upload, a short JSON object.
const maxUploadReply = 1 << 20

// UploadConfig is what an Uploader is built from.
type UploadConfig struct {
	// Token is the author's upload token, sent in the X-Api-Token header. It is not the API key of Config.
	Token string
	// BaseURL is the origin of the game's upload API, such as https://minecraft.curseforge.com; it has no default
	// because each game answers on its own host.
	BaseURL string
	// HTTP sends every request; nil means http.DefaultClient. Redirects to another origin are refused so the token
	// stays with the upload host.
	HTTP *http.Client
	// UserAgent is sent with every request when set.
	UserAgent string
}

// Uploader publishes a file to a CurseForge project through the author upload API.
type Uploader struct {
	token, base, agent string
	hc                 *http.Client
}

// NewUploader builds an Uploader.
func NewUploader(cfg UploadConfig) *Uploader {
	return &Uploader{
		token: strings.TrimSpace(cfg.Token),
		base:  strings.TrimRight(cfg.BaseURL, "/"),
		agent: cfg.UserAgent,
		hc:    httpkit.WithRedirect(cfg.HTTP, httpkit.SameOrigin),
	}
}

// AuthorFilesURL is the author-site page that lists a project's files.
func AuthorFilesURL(projectID int) string {
	return "https://authors.curseforge.com/#/projects/" + strconv.Itoa(projectID) + "/files"
}

// Upload sends the zip at path to the project as a release named displayName, with changelog as markdown, and returns
// the new file's id. A 401 or 403 is a *httpkit.StatusError that Denied matches, and a 429 a *httpkit.RateLimitError.
func (u *Uploader) Upload(ctx context.Context, path string, projectID int, displayName, changelog string) (int, error) {
	if u.token == "" {
		return 0, errors.New("curseforge: no upload token")
	}
	if u.base == "" {
		return 0, errors.New("curseforge: no upload host")
	}
	f, err := safefs.OpenRegular(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	meta, err := json.Marshal(map[string]any{
		"changelog": changelog, "changelogType": "markdown", "releaseType": "release", "displayName": displayName,
	})
	if err != nil {
		return 0, err
	}
	// The file streams from disk through a pipe rather than being held in memory beside its encoding.
	pr, pw := io.Pipe()
	form := multipart.NewWriter(pw)
	go func() {
		err := form.WriteField("metadata", string(meta))
		if err == nil {
			h := textproto.MIMEHeader{}
			h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filepath.Base(path)))
			h.Set("Content-Type", "application/zip")
			var part io.Writer
			if part, err = form.CreatePart(h); err == nil {
				_, err = io.Copy(part, f)
			}
		}
		if err == nil {
			err = form.Close()
		}
		_ = pw.CloseWithError(err)
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.base+"/api/projects/"+strconv.Itoa(projectID)+"/upload-file", pr)
	if err != nil {
		_ = pr.Close()
		return 0, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Api-Token", u.token)
	if u.agent != "" {
		req.Header.Set("User-Agent", u.agent)
	}
	reply, err := httpkit.DoCapped(u.hc, req, maxUploadReply, rateLimited)
	// A refused request leaves the copy above blocked on the pipe until it is closed.
	_ = pr.CloseWithError(err)
	if err != nil {
		return 0, err
	}
	var done struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(reply, &done); err != nil || done.ID <= 0 {
		return 0, errors.New("curseforge: accepted the file but named no file id")
	}
	return done.ID, nil
}
