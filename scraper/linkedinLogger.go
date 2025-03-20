package scraper

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
	"net/http"
	"encoding/csv"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/target"
	
)

// Start Chrome with remote debugging
func StartChrome() error {
	linkedInURL := "https://www.linkedin.com/"

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", "start", "chrome", "--remote-debugging-port=9222", "--profile-directory=Profile 9", linkedInURL)
	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("google-chrome", "--remote-debugging-port=9222", "--profile-directory=Profile 9", linkedInURL)
	} else if runtime.GOOS == "darwin" {
		cmd = exec.Command("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "--remote-debugging-port=9222", "--profile-directory=Profile 9", linkedInURL)
	} else {
		return fmt.Errorf("unsupported OS")
	}

	return cmd.Start()
}


// Find and click the apply button
func ProcessJobApplication(ctx context.Context, jobTitle, jobLink string) ([]string, error) {
	var capturedURLs []string
	var newTabID target.ID

	// Capture current tabs before clicking apply
	initialTabs, err := chromedp.Targets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial open tabs: %v", err)
	}
	existingTabs := make(map[target.ID]struct{})
	for _, t := range initialTabs {
		existingTabs[t.TargetID] = struct{}{}
	}

	// Navigate to job link and click apply button
	err = chromedp.Run(ctx,
		chromedp.Navigate(jobLink),
		chromedp.Sleep(5*time.Second),
		chromedp.Click(`div.jobs-apply-button--top-card button`, chromedp.NodeVisible),
		chromedp.Sleep(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to locate/click apply button: %v", err)
	}

	// Get all open tabs again
	newTabs, err := chromedp.Targets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated open tabs: %v", err)
	}

	// Identify the newly opened application tab
	for _, t := range newTabs {
		if _, exists := existingTabs[t.TargetID]; !exists && t.Type == "page" && t.URL != "" && !strings.Contains(t.URL, "linkedin.com") {
			capturedURLs = append(capturedURLs, t.URL)

			// Save URL
			if err := StoreApplicationLink(jobTitle, t.URL); err != nil {
				log.Printf("❌ Error saving URL to file: %v\n", err)
			}

			fmt.Println("✅ Captured application page:", t.URL)
			newTabID = t.TargetID
			break
		}
	}

	// Close only the newly opened tab
	if newTabID != "" {
		tabCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(newTabID))
		defer cancel()

		err := chromedp.Run(tabCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			return target.CloseTarget(newTabID).Do(ctx)
		}))
		if err != nil {
			log.Printf("❌ Error closing tab: %v\n", err)
		} else {
			fmt.Println("✅ Successfully closed application page tab:", newTabID)
		}
	}

	return capturedURLs, nil
}

// Save the final application URL
func StoreApplicationLink(title, link string) error {
	file, err := os.OpenFile("storage/applications.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

// Process multiple job links
func ProcessJobLinks(ctx context.Context, jobLinks map[string][]string) error {
    ctx, cancel := chromedp.NewContext(ctx)
    defer cancel()

    err := chromedp.Run(ctx,
        chromedp.Navigate("https://www.linkedin.com/search/"),
        chromedp.Sleep(3*time.Second),
    )
    if err != nil {
        return fmt.Errorf("failed to load LinkedIn feed: %v", err)
    }
    fmt.Println("✅ LinkedIn session initiated!")

    for title, links := range jobLinks {
        fmt.Printf("📌 Processing jobs for: %s\n", title)
        for _, jobLink := range links {
            fmt.Println("🔗 Processing:", jobLink)

            _, err := ProcessJobApplication(ctx, title, jobLink) // Fix: Pass title
            if err != nil {
                log.Printf("❌ Error processing job %s: %v\n", title, err)
                continue
            }
        }
    }
    return nil
}


// Read job links from a file
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

// Main function to trigger automation
func LoginLinkedInHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("🚀 Starting LinkedIn job application automation...")

	// Start Chrome
	if err := StartChrome(); err != nil {
		log.Fatalf("❌ Failed to start Chrome: %v", err)
	}
	fmt.Println("✅ Chrome launched successfully.")

	// Wait to ensure Chrome is running
	time.Sleep(5 * time.Second)

	// Load job links from CSV file
	jobLinks, err := LoadJobLinks("storage/job_links.csv")
	if err != nil {
		log.Fatalf("❌ Failed to load job links: %v", err)
	}
	fmt.Printf("✅ Loaded %d job titles with links.\n", len(jobLinks))

	// Create ChromeDP context
	ctx, cancel := chromedp.NewRemoteAllocator(context.Background(), "http://localhost:9222")
	defer cancel()

	// Process all job links correctly
	if err := ProcessJobLinks(ctx, jobLinks); err != nil { // Fix: Pass full jobLinks map
		log.Fatalf("❌ Error processing job links: %v", err)
	}

	fmt.Println("✅ Job application automation completed.")
}
