package salonserver

import (
	"github.com/araquach/salonserver/cmd/db"
	"github.com/araquach/salonserver/cmd/handlers"
	"github.com/araquach/salonserver/cmd/migrations"
	"github.com/araquach/salonserver/cmd/routes"
	"github.com/araquach/salonserver/cmd/shared"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
)

var (
	salon int
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func Serve(s int) {
	dsn := os.Getenv("DATABASE_URL")
	db.DBInit(dsn)

	shared.Salon = s // Set global salon ID

	if salon == 2 {
		migrations.Migrate()
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	handlers.TplIndex = template.Must(template.ParseFiles("index.gohtml"))

	routes.Router()

	log.Printf("Starting server on %s", port)
	http.ListenAndServe(":"+port, forceSsl(&routes.R))

	log.Printf("Starting server on: http://localhost:%s", port)
}

func forceSsl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("GO_ENV") == "production" {
			if r.Header.Get("x-forwarded-proto") != "https" {
				sslUrl := "https://" + r.Host + r.RequestURI
				http.Redirect(w, r, sslUrl, http.StatusTemporaryRedirect)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
