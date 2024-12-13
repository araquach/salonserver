package routes

import "github.com/araquach/salonserver/cmd/handlers"

func priceCalcRoutes() {
	R.HandleFunc("/api/salons", handlers.ApiSalons).Methods("GET")
	R.HandleFunc("/api/stylists", handlers.ApiStylists).Methods("GET")
	R.HandleFunc("/api/levels", handlers.ApiLevels).Methods("GET")
	R.HandleFunc("/api/services", handlers.ApiServices).Methods("GET")
	R.HandleFunc("/api/get-quote-details/{link}", handlers.ApiGetQuoteDetails).Methods("GET")
	R.HandleFunc("/prices/api/save-quote-details", handlers.ApiSaveQuoteDetails).Methods("POST")
}
