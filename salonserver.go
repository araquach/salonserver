package salonserver

import (
	"flag"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
)

var (
	tpl   *template.Template
	salon int
)

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}
}

func Serve(s int) {
	var err error
	var dir string

	dsn := os.Getenv("DATABASE_URL")
	DBInit(dsn)

	salon = s

	if salon == 2 {
		Migrate()
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	tpl = template.Must(template.ParseFiles(
		"index.gohtml"))
	if err != nil {
		panic(err)
	}

	flag.StringVar(&dir, "dir", "dist", "the directory to serve files from")
	flag.Parse()
	r := mux.NewRouter()

	r.PathPrefix("/dist/").Handler(http.StripPrefix("/dist/", http.FileServer(http.Dir(dir))))
	r.HandleFunc("/api/team", apiTeam)
	r.HandleFunc("/api/team/{slug}", apiTeamMember)
	r.HandleFunc("/api/sendMessage", apiSendMessage)
	r.HandleFunc("/api/joinus", apiJoinus)
	r.HandleFunc("/api/joinus-applicants/{status}", apiJoinusApplicants).Methods("GET")
	r.HandleFunc("/api/joinus-applicant/{id}", apiJoinusApplicant).Methods("GET")
	r.HandleFunc("/api/joinus-applicant/{id}", apiJoinUsApplicantUpdate).Methods("PATCH")
	r.HandleFunc("/api/joinus-email-response", apiJoinUsEmailer).Methods("PATCH")
	r.HandleFunc("/api/joinus-update-role/{id}", apiJoinusUpdateRole).Methods("PATCH")
	r.HandleFunc("/api/delete-applicant/{id}", apiDeleteApplicant).Methods("DELETE")

	r.HandleFunc("/api/models", apiModel)
	r.HandleFunc("/api/reviews/{tm}", apiReviews)
	r.HandleFunc("/api/booking-request", apiBookingRequest)
	r.HandleFunc("/api/blog-post/{slug}", apiBlogPost).Methods("GET")
	r.HandleFunc("/api/blog-posts", apiBlogPosts).Methods("GET")
	r.HandleFunc("/api/news-items", apiNewsItems).Methods("GET")
	r.HandleFunc("/api/open-evening", apiOpenEvening).Methods("POST")
	r.HandleFunc("/api/feedback", apiFeedbackResult).Methods("POST")
	r.HandleFunc("/api/store-data", apiStoreData).Methods("GET")
	// priceCalc API
	r.HandleFunc("/api/salons", apiSalons).Methods("GET")
	r.HandleFunc("/api/stylists", apiStylists).Methods("GET")
	r.HandleFunc("/api/levels", apiLevels).Methods("GET")
	r.HandleFunc("/api/services", apiServices).Methods("GET")
	r.HandleFunc("/api/get-quote-details/{link}", apiGetQuoteDetails).Methods("GET")
	r.HandleFunc("/prices/api/save-quote-details", apiSaveQuoteDetails).Methods("POST")

	r.HandleFunc("/{category}/{name}", home)
	r.HandleFunc("/{name}", home)
	r.HandleFunc("/", home)

	log.Printf("Starting server on: http://localhost:%s", port)

	http.ListenAndServe(":"+port, forceSsl(r))
}
