package Linkedin

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"
	"net/http"
	"encoding/json"

	
	//"path/filepath"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/target"
	
)

var chromeCmd *exec.Cmd // Global variable to track the process

func StartChrome() (*exec.Cmd, error) {
	linkedInURL := "https://www.linkedin.com/"
	chromePath := "chromium" // Path to Chromium executable

	if runtime.GOOS == "linux" {
		// Launch Chromium with Profile 2
		chromeCmd := exec.Command(chromePath, "--remote-debugging-port=9222", "--profile-directory=Profile 2", linkedInURL)
		err := chromeCmd.Start()
		if err != nil {
			return nil, fmt.Errorf("failed to start Chromium: %w", err)
		}
		return chromeCmd, nil
	}
	return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
}


func LoginLinkedInHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("🚀 Starting LinkedIn job application automation...")

	// Start Chrome (Chromium) with remote debugging
	linkedInURL := "https://www.linkedin.com/"
	chromePath := "chromium" // Path to Chromium executable

	var chromeCmd *exec.Cmd // Global variable to track the process

	if runtime.GOOS == "linux" {
		// Launch Chromium with Profile 2
		chromeCmd = exec.Command(chromePath, "--remote-debugging-port=9222", "--profile-directory=Profile 2", linkedInURL)
	} else {
		log.Fatalf("❌ Unsupported OS: %s", runtime.GOOS)
	}
	err := chromeCmd.Start()
	if err != nil {
		log.Fatalf("❌ Failed to start Chromium: %v", err)
	}

	fmt.Println("✅ Chrome launched successfully.")
	time.Sleep(5 * time.Second) // Wait for Chrome to start and be ready for remote debugging

	// Load job links from CSV file
	jobLinks, err := LoadJobLinks("storage/Linkedin_jobs.csv") // Relative to JSE root
	if err != nil {
		log.Printf("❌ Failed to load job links: %v\n", err)
		http.Error(w, "Failed to load job links", http.StatusInternalServerError)
		return
	}
	fmt.Printf("✅ Loaded %d job titles with links.\n", len(jobLinks))

	// Initialize CSV files
	err = InitializeCSVFiles() // Initialize the CSV files with headers
	if err != nil {
		log.Fatalf("❌ Failed to initialize CSV files: %v", err)
	}

	// Create a context for chromedp using the running browser instance
	allocatorCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), "http://localhost:9222")
	defer cancel()
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	// Process all job links correctly
	for title, links := range jobLinks {
		fmt.Printf("📌 Processing jobs for: %s\n", title)
		for _, jobLink := range links {
			fmt.Println("🔗 Processing:", jobLink)

			// Capture initial tabs
			initialTabs, err := chromedp.Targets(ctx)
			if err != nil {
				log.Printf("❌ Failed to get initial open tabs: %v", err)
				StoreFailedJob(title, jobLink, "Failed to get initial open tabs")
				continue
			}

			existingTabs := make(map[target.ID]struct{})
			for _, t := range initialTabs {
				existingTabs[t.TargetID] = struct{}{}
			}

			// Use the new navigateAndClickApply function
			err = navigateAndClickApply(ctx, title, jobLink)
			if err != nil {
				// If an error occurs in navigateAndClickApply, skip to the next job
				continue
			}

			// Use captureAndCloseNewTab to capture URLs and close new tabs
			capturedURLs, err := captureAndCloseNewTab(ctx, title, existingTabs)
			if err != nil {
				// If there was an error, log and continue
				StoreFailedJob(title, jobLink, "Failed to capture and close new tab")
				continue
			}

			// Log the captured URLs
			for _, url := range capturedURLs {
				fmt.Printf("Captured application link: %s\n", url)
			}
		}
	}

	// Stop Chromium browser
	if chromeCmd != nil {
		fmt.Println("🛑 Closing Chromium...")
		if err := chromeCmd.Process.Kill(); err != nil {
			fmt.Printf("⚠️ Failed to close Chromium: %v\n", err)
		} else {
			fmt.Println("✅ Chromium closed successfully.")
		}
	}

	// Send a success message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Job links saved in job_links.csv"})
}
