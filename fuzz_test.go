package curseforge

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

type canned []byte

func (c canned) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(c)), Header: http.Header{}}, nil
}

func TestFuzzDecodersReachTheTransport(t *testing.T) {
	var n int
	rt := roundTripCount{n: &n, RoundTripper: canned([]byte(`{"data":null}`))}
	c := New(Config{APIKey: "k", BaseURL: "http://127.0.0.1", HTTP: &http.Client{Transport: rt}})
	_, _ = c.Mod(t.Context(), 1)
	if n == 0 {
		t.Fatal("RoundTrip never ran")
	}
}

type roundTripCount struct {
	n *int
	http.RoundTripper
}

func (r roundTripCount) RoundTrip(req *http.Request) (*http.Response, error) {
	*r.n++
	return r.RoundTripper.RoundTrip(req)
}

// FuzzDecoders holds that no answer body makes any call panic.
func FuzzDecoders(f *testing.F) {
	f.Add([]byte(`{"data":[{"id":1,"name":"T","latestFiles":[{"id":2,"isAvailable":true,"hashes":[{"value":"ab","algo":1}],"dependencies":[{"modId":3,"relationType":3}]}]}],"pagination":{"totalCount":1}}`))
	f.Add([]byte(`{"data":{"id":1,"allowModDistribution":false}}`))
	f.Add([]byte(`{"data":"<p>x</p>"}`))
	f.Add([]byte(`{"data":null}`))
	f.Add([]byte(`{`))
	f.Fuzz(func(t *testing.T, body []byte) {
		c := New(Config{APIKey: "k", BaseURL: "http://127.0.0.1", HTTP: &http.Client{Transport: canned(body)}})
		ctx := t.Context()
		_, _ = c.Search(ctx, SearchQuery{GameID: 1})
		m, _ := c.Mod(ctx, 1)
		_ = m.Restricted()
		mods, _ := c.Mods(ctx, []int{1})
		for _, m := range mods {
			Newest(m.LatestFiles, nil, ReleaseAlpha)
		}
		files, _ := c.Files(ctx, 1, FilesQuery{})
		for _, f := range files.Files {
			_, _ = f.SHA1(), f.Required()
		}
		_, _ = c.File(ctx, 1, 2)
		_, _ = c.Description(ctx, 1)
		_, _ = c.Changelog(ctx, 1, 2)
		_, _ = c.DownloadURL(ctx, 1, 2)
		_, _ = c.Classes(ctx, 1)
		_, _ = c.Categories(ctx, 1, 2)
	})
}

// FuzzModUnmarshalJSON holds that a decoded Mod panics on nothing and that Restricted
// and Available match the documented flag rules.
func FuzzModUnmarshalJSON(f *testing.F) {
	f.Add([]byte(`{"id":1,"allowModDistribution":false}`))
	f.Add([]byte(`{"id":1,"isAvailable":false}`))
	f.Add([]byte(`{"id":1,"isAvailable":true,"allowModDistribution":true}`))
	f.Add([]byte(`{"id":1}`))
	f.Add([]byte(`{"data":null}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[]`))
	f.Fuzz(func(t *testing.T, b []byte) {
		var m Mod
		if err := m.UnmarshalJSON(b); err != nil {
			return
		}
		if m.Restricted() && (m.AllowModDistribution == nil || *m.AllowModDistribution) {
			t.Fatalf("Restricted with allow=%v", m.AllowModDistribution)
		}
		if !m.Available() && (m.available == nil || *m.available) {
			t.Fatalf("unavailable without isAvailable:false")
		}
	})
}
