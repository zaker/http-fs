package main

import (
	"crypto/sha256"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"strings"
)

//go:embed static
var staticFiles embed.FS

// computeETag hashes all embedded files once at startup
func computeETag(fsys fs.FS) string {
	h := sha256.New()
	_ = fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, path)
		if err == nil {
			h.Write([]byte(path))
			h.Write(data)
		}
		return nil
	})
	return fmt.Sprintf(`"%x"`, h.Sum(nil)[:16]) // Weak or strong ETag quotes required
}

func ShannonEntropy(s string) (perChar float64, total float64) {
	if len(s) == 0 {
		return 0, 0
	}

	// Count frequency of each byte (use rune if checking multi-byte UTF-8)
	counts := make(map[byte]int)
	for i := 0; i < len(s); i++ {
		counts[s[i]]++
	}

	length := float64(len(s))
	for _, count := range counts {
		p := float64(count) / length
		perChar -= p * math.Log2(p)
	}

	return perChar, perChar * length
}

func IsHighEntropyPath(s string) bool {
	perChar, total := ShannonEntropy(s)
	fmt.Printf("string: %v, perChar: %v, total: %v\n", s, perChar, total)
	return perChar >= 3.5 && total >= 60
}

func staticHandler() (http.Handler, error) {
	files, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}
	etag := computeETag(files)
	fileServer := http.FileServer(http.FS(files))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Immutable caching for fingerprinted assets (e.g., /assets/main.8f92a.js)
		if IsHighEntropyPath(r.URL.Path) {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fileServer.ServeHTTP(w, r)
			return
		}

		// 2. ETag revalidation for index.html and unhashed files
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "no-cache") // Forces browser to validate ETag before using cache

		if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		fileServer.ServeHTTP(w, r)
	}), nil
}

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	handler, err := staticHandler()
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.ListenAndServe(*addr, handler))
}
