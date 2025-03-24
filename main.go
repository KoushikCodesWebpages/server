package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
	"io/ioutil"
	
	"job_scraper/scraper" 
	"job_scraper/scraper/Linkedin"
	"syscall"
)

func suppressLogs() {
	log.SetOutput(ioutil.Discard) // Disables all logs
}



func main() {
	suppressLogs()

	// Set up a channel to listen for an interrupt signal (Ctrl+C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Create a new HTTP request multiplexer (mux)
	mux := http.NewServeMux()

	// Define the routes and their handlers
	mux.HandleFunc("/joblistings", Linkedin.JobListingsHandler)  
	// Linkedin Job listings route
	mux.HandleFunc("/loginlinkedin", Linkedin.LoginLinkedInHandler) 
    // Database upload route
	mux.HandleFunc("/uploaddb", Linkedin.PostDBHandler) 
	//Final Automation
	mux.HandleFunc("/linkedinautomation", Linkedin.LinkedInHandler)           // Automation route

	// Enable CORS support
	handler := enableCors(mux)

	// Define the server port
	port := ":5000"
	server := &http.Server{
		Addr:    port,
		Handler: handler,
	}

	// Start the HTTP server in another goroutine
	go func() {
		fmt.Printf("🚀 Server is running on http://localhost%s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server failed: %v", err)
		}
	}()

	// Wait for an interrupt signal to gracefully shut down the server and the browser
	<-quit

	// Gracefully shut down the server
	fmt.Println("\n🚪 Shutting down the server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server shutdown failed: %v", err)
	}

	// Terminate the browser (chromedp or any other process you are running)
	fmt.Println("🖥️ Terminating browser...")
	if err := scraper.TerminateBrowser(); err != nil {
		log.Printf("❌ Failed to terminate the browser: %v", err)
	} else {
		fmt.Println("✅ Browser terminated successfully.")
	}

	fmt.Println("✅ Server shut down gracefully.")
}

// enableCors adds CORS headers to allow cross-origin requests
func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := "http://localhost:3000" // Frontend server URL or adjust if needed
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Respond to OPTIONS pre-flight request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Serve the request to the next handler
		next.ServeHTTP(w, r)
	})
}
