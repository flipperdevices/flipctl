package main

import (
	"flag"
	"github.com/flipperdevices/flipctl/internal/api"
	"github.com/flipperdevices/flipctl/internal/plugin"
	"log"
	"log/slog"
	"net/http"
	"os"
)

var version = "dev"

func main() {
	addr := flag.String("addr", ":8080", "")
	plug := flag.String("plugins", "plugins", "")
	web := flag.String("web", "web/dist", "")
	flag.Parse()
	apps, err := plugin.LoadDir(*plug)
	if err != nil {
		log.Fatal(err)
	}
	slog.Info("flipctld starting", "version", version, "apps", len(apps))
	var st http.Handler
	if _, e := os.Stat(*web); e == nil {
		st = http.FileServer(http.Dir(*web))
	}
	log.Fatal(http.ListenAndServe(*addr, api.New(apps, st).Handler()))
}
