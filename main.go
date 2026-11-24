package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func staticHandler() (http.Handler, error) {
	files, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(files)), nil
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
