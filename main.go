package main

import (
	"fmt"
	"log"
	"net/http"
	"io/ioutil"

	"job_scraper/scraper" // Correct import path
)

func suppressLogs() {
	log.SetOutput(ioutil.Discard) // Disables all logs
}

func main() {
	suppressLogs()

	mux := http.NewServeMux()

	mux.HandleFunc("/joblistings", scraper.JobListingsHandler)
	mux.HandleFunc("/loginlinkedin", scraper.LoginLinkedInHandler)
	mux.HandleFunc("/uploaddb", scraper.PostDBHandler)

	handler := enableCors(mux)

	port := ":5000"
	fmt.Printf("🚀 Server is running on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, handler))
}

func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := "http://localhost:3000"
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
