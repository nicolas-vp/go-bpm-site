// Команда serve - локальный просмотр собранного сайта. /api/ и /ui/ проксируются в демо go-bpm, как nginx на сервере:
//
//	ssh -L 8080:127.0.0.1:8080 root@213.239.158.215   # демо с сервера
//	go run ./cmd/serve -dir dist -demo http://localhost:8080
package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dir := flag.String("dir", "dist", "собранный сайт")
	addr := flag.String("addr", "localhost:8000", "адрес")
	demo := flag.String("demo", "http://localhost:8080", "демо go-bpm для /api/ и /ui/")
	flag.Parse()

	target, err := url.Parse(*demo)
	if err != nil {
		log.Fatal(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	files := http.FileServer(http.Dir(*dir))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ui" || strings.HasPrefix(r.URL.Path, "/ui/") {
			proxy.ServeHTTP(w, r)
			return
		}
		if _, err := os.Stat(filepath.Join(*dir, filepath.FromSlash(r.URL.Path))); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			http.ServeFile(w, r, filepath.Join(*dir, "404.html"))
			return
		}
		files.ServeHTTP(w, r)
	})
	log.Printf("сайт: http://%s/, демо: %s", *addr, *demo)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
