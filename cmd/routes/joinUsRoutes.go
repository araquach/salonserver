package routes

import "github.com/araquach/salonserver/cmd/handlers"

func apiJoinUsRoutes() {
	R.HandleFunc("/api/sendMessage", handlers.ApiSendMessage)
	R.HandleFunc("/api/joinus", handlers.ApiJoinus)
	R.HandleFunc("/api/joinus-applicants/{status}", handlers.ApiJoinusApplicants).Methods("GET")
	R.HandleFunc("/api/joinus-applicant/{id}", handlers.ApiJoinusApplicant).Methods("GET")
	R.HandleFunc("/api/joinus-applicant/{id}", handlers.ApiJoinUsApplicantUpdate).Methods("PATCH")
	R.HandleFunc("/api/joinus-email-response", handlers.ApiJoinUsEmailer).Methods("PATCH")
	R.HandleFunc("/api/joinus-update-role/{id}", handlers.ApiJoinusUpdateRole).Methods("PATCH")
	R.HandleFunc("/api/delete-applicant/{id}", handlers.ApiDeleteApplicant).Methods("DELETE")
}
