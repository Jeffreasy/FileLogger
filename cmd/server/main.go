package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"filesystem-logger/internal/api"
	"filesystem-logger/web/handlers"
)

func main() {
	router := mux.NewRouter()

	// Static files
	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/",
			http.FileServer(http.Dir("web/static"))))

	// API routes
	router.HandleFunc("/api/scan", api.StartScan).Methods("POST")
	router.HandleFunc("/api/status", api.GetStatus).Methods("GET")
	router.HandleFunc("/api/ws", api.WebSocketHandler)

	// Web routes
	router.HandleFunc("/", handlers.HomePage)
	router.HandleFunc("/scan", handlers.ScanPage)
	router.HandleFunc("/results", handlers.ResultsPage)

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
