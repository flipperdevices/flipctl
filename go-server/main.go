package main

import (
	"archive/zip"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

type zipSPA struct {
	inner fs.FS
}

func (z *zipSPA) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}
	if _, err := fs.Stat(z.inner, path); err != nil {
		r = r.Clone(r.Context())
		r.URL.Path = "/"
	}
	http.FileServer(http.FS(z.inner)).ServeHTTP(w, r)
}

func mustZip(path string) *zipSPA {
	r, err := zip.OpenReader(path)
	if err != nil {
		log.Fatalf("open %q: %v", path, err)
	}
	return &zipSPA{inner: r}
}

func main() {
	go func() {
		log.Println("main     → http://localhost:5173")
		log.Fatal(http.ListenAndServe(":5173", mustZip("../apps/main.zip")))
	}()

	log.Println("ifconfig → http://localhost:5174")
	log.Fatal(http.ListenAndServe(":5174", mustZip("../apps/ifconfig.zip")))
}
