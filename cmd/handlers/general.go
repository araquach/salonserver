package handlers

import (
	"html/template"
	"math/rand"
	"net/http"
	"time"
)

var (
	TplIndex *template.Template
)

func Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	rand.Seed(time.Now().UnixNano())

	v := string(rand.Intn(30))

	if err := TplIndex.Execute(w, v); err != nil {
		panic(err)
	}
}
