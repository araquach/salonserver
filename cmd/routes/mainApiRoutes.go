package routes

import "github.com/araquach/salonserver/cmd/handlers"

func mainApiRoutes() {
	R.HandleFunc("/api/team", handlers.ApiTeam)
	R.HandleFunc("/api/team/{slug}", handlers.ApiTeamMember)
	R.HandleFunc("/api/models", handlers.ApiModel)
	R.HandleFunc("/api/reviews/{tm}", handlers.ApiReviews)
	R.HandleFunc("/api/booking-request", handlers.ApiBookingRequest)
	R.HandleFunc("/api/blog-post/{slug}", handlers.ApiBlogPost).Methods("GET")
	R.HandleFunc("/api/blog-posts", handlers.ApiBlogPosts).Methods("GET")
	R.HandleFunc("/api/news-items", handlers.ApiNewsItems).Methods("GET")
	R.HandleFunc("/api/open-evening", handlers.ApiOpenEvening).Methods("POST")
	R.HandleFunc("/api/feedback", handlers.ApiFeedbackResult).Methods("POST")
	R.HandleFunc("/api/store-data", handlers.ApiStoreData).Methods("GET")
}
