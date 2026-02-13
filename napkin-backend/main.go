package main

import (
	"log"
	"net/http"

	"napkin-backend/lib/handlers"
	"napkin-backend/lib/middleware"
	"napkin-backend/lib/static"
)

func main() {
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("GET /health", handlers.HealthHandler)
	mux.HandleFunc("GET /api/hello", handlers.HelloHandler)

	// Serve static files in production
	static.SetupStaticHandler(mux)

	// Enable CORS middleware (only needed in development when not using proxy)
	handler := middleware.CORS(mux)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
