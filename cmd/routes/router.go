package routes

import (
	"flag"
	"github.com/araquach/salonserver/cmd/handlers"
	"github.com/gorilla/mux"
	"net/http"
)

var R mux.Router

func Router() {
	var dir string

	flag.StringVar(&dir, "dir", "dist", "the directory to serve files from")
	flag.Parse()

	R.PathPrefix("/dist/").Handler(http.StripPrefix("/dist/", http.FileServer(http.Dir(dir))))

	apiJoinUsRoutes()
	mainApiRoutes()
	priceCalcRoutes()

	R.HandleFunc("/{category}/{name}", handlers.Home)
	R.HandleFunc("/{name}", handlers.Home)
	R.HandleFunc("/", handlers.Home)

	return
}
