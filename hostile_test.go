package curseforge

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-safefs"
)

type answer struct {
	code       int
	retryAfter string
	body       []byte
}

// hostile serves every request with the answer last stored in a. One server serves a whole fuzz run: a server per
// input runs the machine out of ports.
func hostile(t testing.TB, a *atomic.Pointer[answer]) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cur := a.Load()
		w.Header()["Retry-After"] = []string{cur.retryAfter}
		w.WriteHeader(cur.code)
		_, _ = w.Write(cur.body)
	}))
	t.Cleanup(srv.Close)
	return New(Config{APIKey: "k", BaseURL: srv.URL})
}

// FuzzHostileAnswer holds that no status, Retry-After or body makes a call panic, and that a 429 is always a
// *httpkit.RateLimitError whose reset, when set, is not before the Unix epoch.
func FuzzHostileAnswer(f *testing.F) {
	f.Add(429, "99999999999999999999", []byte(`{}`))
	f.Add(429, "Wed, 21 Oct 2015 07:28:00 GMT", []byte(nil))
	f.Add(200, "-1", []byte(strings.Repeat(`{"data":`, 20000)))
	f.Add(200, "", []byte(`{"data":{"id":1,"latestFiles":[{"hashes":[{"algo":99}]}]}}`))
	var a atomic.Pointer[answer]
	a.Store(&answer{code: 200})
	c := hostile(f, &a)
	f.Fuzz(func(t *testing.T, code int, retryAfter string, body []byte) {
		if code < 200 || code > 599 || code/100 == 3 {
			code = 200
		}
		a.Store(&answer{code: code, retryAfter: retryAfter, body: body})
		_, err := c.Mod(t.Context(), 1)
		// net/http refuses a header line with control bytes before the client sees a status.
		malformed := strings.ContainsFunc(retryAfter, func(r rune) bool { return r < ' ' && r != '\t' || r == 0x7f })
		if code == http.StatusTooManyRequests && !malformed {
			re, ok := errors.AsType[*httpkit.RateLimitError](err)
			if !ok {
				t.Fatalf("429 gave %v", err)
			}
			if !re.Reset.IsZero() && re.Reset.Unix() < 0 {
				t.Fatalf("reset %v from Retry-After %q", re.Reset, retryAfter)
			}
		}
		_, _ = c.Search(t.Context(), SearchQuery{GameID: 1})
		_, _ = c.DownloadURL(t.Context(), 1, 2)
	})
}

func TestAnAnswerOverTheCapFails(t *testing.T) {
	t.Parallel()
	var a atomic.Pointer[answer]
	a.Store(&answer{code: http.StatusOK, body: bytes.Repeat([]byte(" "), maxBody+1)})
	if _, err := hostile(t, &a).Mod(t.Context(), 1); !errors.Is(err, safefs.ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

func TestALyingContentLengthFails(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, rw, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = rw.WriteString("HTTP/1.1 200 OK\r\nContent-Length: 1099511627776\r\n\r\n{\"data\":{}}")
		_ = rw.Flush()
	}))
	t.Cleanup(srv.Close)
	if _, err := New(Config{APIKey: "k", BaseURL: srv.URL}).Mod(t.Context(), 1); err == nil {
		t.Fatal("a body cut short of its Content-Length was accepted")
	}
}
