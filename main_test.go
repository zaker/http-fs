package main

import (
	"encoding/hex"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestComputeETag(t *testing.T) {
	files := fstest.MapFS{
		"index.html":    {Data: []byte("hello")},
		"assets/app.js": {Data: []byte("app")},
	}
	tag := computeETag(files)
	if len(tag) != 34 || tag[0] != '"' || tag[len(tag)-1] != '"' {
		t.Fatalf("ETag = %q, want a quoted 16-byte hash", tag)
	}
	if _, err := hex.DecodeString(tag[1 : len(tag)-1]); err != nil {
		t.Fatalf("ETag = %q: invalid hex: %v", tag, err)
	}
	if got := computeETag(files); got != tag {
		t.Errorf("ETag changed for the same files: %q != %q", got, tag)
	}

	for _, tc := range []struct {
		name  string
		files fstest.MapFS
	}{
		{"changed content", fstest.MapFS{
			"index.html":    {Data: []byte("goodbye")},
			"assets/app.js": {Data: []byte("app")},
		}},
		{"changed path", fstest.MapFS{
			"index.html":     {Data: []byte("hello")},
			"assets/main.js": {Data: []byte("app")},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := computeETag(tc.files); got == tag {
				t.Errorf("ETag did not change: %q", got)
			}
		})
	}
}

func TestShannonEntropy(t *testing.T) {
	for _, tc := range []struct {
		input          string
		perChar, total float64
	}{
		{"", 0, 0},
		{"aaaa", 0, 0},
		{"aabb", 1, 4},
		{"abcd", 2, 8},
	} {
		t.Run(tc.input, func(t *testing.T) {
			perChar, total := ShannonEntropy(tc.input)
			if math.Abs(perChar-tc.perChar) > 1e-9 || math.Abs(total-tc.total) > 1e-9 {
				t.Errorf("ShannonEntropy(%q) = (%v, %v), want (%v, %v)", tc.input, perChar, total, tc.perChar, tc.total)
			}
		})
	}
}

func TestIsHighEntropyPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		want bool
	}{
		{"index", "/index.html", false},
		{"unfingerprinted asset", "/assets/main.js", false},
		{"short fingerprint", "/assets/main.8f92a.js", true},
		{"fingerprinted asset", "/assets/main.0123456789abcdef0123456789abcdef.js", true},
		{"long repetitive path", "/" + strings.Repeat("a", 160), false},
		{"below total threshold", "0011223456789abc", false},
		{"at total threshold", "001123456789abcd", true},
		{"fingerprinted js file", "/main-5NT4ESZH.js", true},
		{"below per-character threshold", strings.Repeat("001122334456789a", 3), false},
		{"at per-character threshold", strings.Repeat("00112233456789ab", 3), true},
		{"high total but low per-character entropy", strings.Repeat("01", 128), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsHighEntropyPath(tc.path); got != tc.want {
				t.Errorf("IsHighEntropyPath(%q) = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}

func TestStaticHandler(t *testing.T) {
	handler, err := staticHandler()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("serves embedded index", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if !strings.Contains(recorder.Body.String(), "<h1>http-fs</h1>") {
			t.Fatalf("unexpected body: %q", recorder.Body.String())
		}
		if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "text/html") {
			t.Fatalf("unexpected content type: %q", recorder.Header().Get("Content-Type"))
		}
	})

	t.Run("missing file", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/missing.txt", nil))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
		}
	})

	t.Run("ETag revalidation", func(t *testing.T) {
		initial := httptest.NewRecorder()
		handler.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/", nil))
		tag := initial.Header().Get("ETag")
		if tag == "" {
			t.Fatal("initial response has no ETag")
		}
		if got := initial.Header().Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("Cache-Control = %q, want no-cache", got)
		}

		for _, tc := range []struct {
			name, match string
			status      int
		}{
			{"matching", tag, http.StatusNotModified},
			{"matching in list", `"other", ` + tag, http.StatusNotModified},
			{"nonmatching", `"other"`, http.StatusOK},
		} {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("If-None-Match", tc.match)
				recorder := httptest.NewRecorder()
				handler.ServeHTTP(recorder, req)
				if recorder.Code != tc.status {
					t.Errorf("status = %d, want %d", recorder.Code, tc.status)
				}
				if got := recorder.Header().Get("ETag"); got != tag {
					t.Errorf("ETag = %q, want %q", got, tag)
				}
				if got := recorder.Header().Get("Cache-Control"); got != "no-cache" {
					t.Errorf("Cache-Control = %q, want no-cache", got)
				}
				if tc.status == http.StatusNotModified && recorder.Body.Len() != 0 {
					t.Errorf("304 response body = %q, want empty", recorder.Body.String())
				}
				if tc.status == http.StatusOK && recorder.Body.String() != initial.Body.String() {
					t.Errorf("response body = %q, want %q", recorder.Body.String(), initial.Body.String())
				}
			})
		}
	})

	t.Run("high entropy path uses immutable caching", func(t *testing.T) {
		// FileServer cleans the path to "/", which serves the embedded index.
		path := "/assets/main.0123456789abcdef0123456789abcdef.js/../../"
		initial := httptest.NewRecorder()
		handler.ServeHTTP(initial, httptest.NewRequest(http.MethodGet, "/", nil))
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("If-None-Match", initial.Header().Get("ETag"))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if got := recorder.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
			t.Errorf("Cache-Control = %q, want immutable caching", got)
		}
		if got := recorder.Header().Get("ETag"); got != "" {
			t.Errorf("ETag = %q, want none", got)
		}
		if !strings.Contains(recorder.Body.String(), "<h1>http-fs</h1>") {
			t.Errorf("unexpected body: %q", recorder.Body.String())
		}
	})
}
