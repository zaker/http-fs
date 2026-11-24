package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
}
