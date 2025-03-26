package Linkedin

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
	"net/http"
	"encoding/csv"
	"encoding/json"
	
	//"path/filepath"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/target"
	
)

// Start Chrome with remote debugging
// Start Chrome with remote debugging (Chromium specifically)
var chromeCmd *exec.Cmd // Global variable to track the process

func StartChrome() (*exec.Cmd, error) {
	linkedInURL := "https://www.linkedin.com/"

	if runtime.GOOS == "windows" {
		chromeCmd = exec.Command("cmd", "/C", "start", "chromium", "--remote-debugging-port=9222", "--profile-directory=Profile 9", linkedInURL)
	} else if runtime.GOOS == "linux" {
		chromeCmd = exec.Command("/snap/bin/chromium", "--remote-debugging-port=9222", "--profile-directory=Profile 1", linkedInURL)
	} else if runtime.GOOS == "darwin" {
		chromeCmd = exec.Command("/Applications/Chromium.app/Contents/MacOS/Chromium", "--remote-debugging-port=9222", "--profile-directory=Profile 9", linkedInURL)
	} else {
		return nil, fmt.Errorf("unsupported OS")
	}

	err := chromeCmd.Start()
	if err != nil {
		return nil, err
	}

	return chromeCmd, nil
}



func LoadJobLinks(filename string) (map[string][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV file: %v", err)
	}

	// Ensure there's at least a header row
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or contains only headers")
	}

	jobLinks := make(map[string][]string) // Map of job title -> []links

	for _, row := range records[1:] { // Skip header row
		if len(row) < 2 {
			continue // Ignore malformed rows
		}
		title, link := row[0], row[1]
		jobLinks[title] = append(jobLinks[title], link)
	}

	return jobLinks, nil
}


// Process multiple job links
func ProcessJobLinks(ctx context.Context, jobLinks map[string][]string) error {
	ctx, cancel := chromedp.NewContext(ctx) // Create new browser session
	defer cancel()

	// Start LinkedIn session
	err := chromedp.Run(ctx,
		chromedp.Navigate("https://www.linkedin.com/search/"),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		return fmt.Errorf("failed to load LinkedIn feed: %v", err)
	}
	fmt.Println("✅ LinkedIn session initiated!")

	// Process each job link
	for title, links := range jobLinks {
		fmt.Printf("📌 Processing jobs for: %s\n", title)
		for _, jobLink := range links {
			fmt.Println("🔗 Processing:", jobLink)

			_, err := ProcessJobApplication(ctx, title, jobLink)
			if err != nil {
				log.Printf("❌ Error processing job %s: %v\n", title, err)
				continue
			}
		}
	}

	// Close all browser tabs after processing
	targets, err := chromedp.Targets(ctx)
	if err != nil {
		log.Printf("❌ Failed to retrieve browser targets: %v", err)
	} else {
		for _, t := range targets {
			if t.Type == "page" { // Ensure we only close pages (tabs)
				err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
					return target.CloseTarget(t.TargetID).Do(ctx)
				}))
				if err != nil {
					log.Printf("❌ Error closing tab: %v\n", err)
				} else {
					fmt.Println("✅ Closed tab:", t.URL)
				}
			}
		}
	}

	// Terminate the browser session completely
	cancel()
	fmt.Println("🚪 Browser session terminated.")

	return nil
}




// ProcessJobApplication is the main function that calls the helper functions
func ProcessJobApplication(ctx context.Context, jobTitle, jobLink string) ([]string, error) {
	// Capture existing open tabs
	initialTabs, err := chromedp.Targets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial open tabs: %v", err)
	}
	existingTabs := make(map[target.ID]struct{})
	for _, t := range initialTabs {
		existingTabs[t.TargetID] = struct{}{}
	}

	// Set up a 15-second timeout
	timer := time.After(15 * time.Second)
	done := make(chan struct{})
	var capturedURLs []string

	go func() {
		defer close(done)

		// Step 1: Navigate and click apply
		err := navigateAndClickApply(ctx, jobTitle, jobLink)
		if err != nil {
			return
		}

		// Step 2: Capture and close the new tab
		capturedURLs, _ = captureAndCloseNewTab(ctx, jobTitle, existingTabs)

		done <- struct{}{}
	}()

	// Ensure process does not exceed 15 seconds
	select {
	case <-timer:
		fmt.Println("⏳ Timeout reached for:", jobTitle)
		StoreFailedJob(jobTitle, jobLink, "Timeout exceeded")
	case <-done:
		fmt.Println("✅ Completed within time for:", jobTitle)
	}

	return capturedURLs, nil
}



func StoreFailedJob(title, link, reason string) error {
	file, err := os.OpenFile("storage/failed_jobs.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write job title, link, and reason for failure
	if err := writer.Write([]string{title, link, reason, time.Now().Format(time.RFC3339)}); err != nil {
		return fmt.Errorf("❌ Failed to write to CSV: %v", err)
	}

	fmt.Printf("⚠️ Stored failed job: %s -> %s (Reason: %s)\n", title, link, reason)
	return nil
}


// Save the final application URL
func StoreApplicationLink(title, link string) error {
	file, err := os.OpenFile("storage/Linkedin_joblinks.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()
	// Write job title and link to CSV
	if err := writer.Write([]string{title, link}); err != nil {
		return fmt.Errorf("❌ Failed to write to CSV: %v", err)
	}

	fmt.Printf("✅ Stored application: %s -> %s\n", title, link)
	return nil
}




// Main function to trigger automation
func LoginLinkedInHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("🚀 Starting LinkedIn job application automation...")

	// Start Chrome (Chromium) with remote debugging
	_, err := StartChrome()
	if err != nil {
		log.Fatalf("❌ Failed to start Chrome: %v", err)
	}

	fmt.Println("✅ Chrome launched successfully.")

	// Wait for Chrome to start and be ready for remote debugging
	time.Sleep(5 * time.Second)
	
	// Load job links from CSV file



	jobLinks, err := LoadJobLinks("storage/Linkedin_jobs.csv") // Relative to JSE root
	if err != nil {
		log.Printf("❌ Failed to load job links: %v\n", err)
		http.Error(w, "Failed to load job links", http.StatusInternalServerError)
		return
	}
	fmt.Printf("✅ Loaded %d job titles with links.\n", len(jobLinks))
	
	if err := InitializeCSVFiles(); err != nil {
		log.Fatalf("❌ Failed to initialize CSV files: %v", err)
	}
	// Connect to the already running Chrome instance using the remote debugger
	allocatorCtx, cancel := chromedp.NewRemoteAllocator(context.Background(), "http://localhost:9222")
	defer cancel()

	// Create a new chromedp context with the allocator context
	ctx, cancel := chromedp.NewContext(allocatorCtx)
	defer cancel()

	// Process all job links correctly
	if err := ProcessJobLinks(ctx, jobLinks); err != nil {
		log.Printf("❌ Error processing job links: %v\n", err)
		http.Error(w, "Error processing job links", http.StatusInternalServerError)
		return
	}
	fmt.Println("✅ Job application automation completed.")
	StopChrome()

	// Send a success message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Job links saved in job_links.csv"})
}



