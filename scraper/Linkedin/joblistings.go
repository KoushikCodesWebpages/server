package Linkedin

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"net/http"
	"time"

	"github.com/chromedp/chromedp"
)

// Job struct to hold job data
type Job struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	IsEasyApply bool   `json:"isEasyApply"` // Add this field
}

// Clean text utility function
func cleanText(text string) string {
	return regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(text), " ")
}

// Construct search URL function
func constructSearchUrl(keywords, location, dateSincePosted string) string {
	return fmt.Sprintf("https://www.linkedin.com/jobs/search?keywords=%s&location=%s&f_TPR=%s&position=1&pageNum=0",
		strings.ReplaceAll(keywords, " ", "%20"),
		strings.ReplaceAll(location, " ", "%20"),
		dateSincePosted,
	)
}

// Fetch job listings for multiple titles and store in CSV
func fetchAndStoreJobs(ctx context.Context, jobTitles []string, location, dateSincePosted string) error {
	file, err := os.Create("storage/Linkedin_jobs.csv")
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	writer.Write([]string{"Title", "Job Link", "Components", "Description"})

	for _, title := range jobTitles {
		searchURL := constructSearchUrl(title, location, dateSincePosted)
		var jobs []Job

		err := chromedp.Run(ctx,
			chromedp.Navigate(searchURL),
			chromedp.Sleep(5*time.Second),
			chromedp.WaitVisible(`.jobs-search__results-list`, chromedp.ByQuery),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('.jobs-search__results-list li')).map(el => ({
				title: "`+title+`",
				link: el.querySelector('.base-card__full-link')?.href || '',
				isEasyApply: el.querySelector('.jobs-apply-button--top-card') !== null // Detect Easy Apply button
			}))`, &jobs),
		)

		if err != nil {
			fmt.Printf("❌ Failed to fetch jobs for %s: %v\n", title, err)
			continue
		}

		// Limit to 5 job links per title (excluding Easy Apply)
		count := 0
		for _, job := range jobs {
			if job.Link != "" && !job.IsEasyApply { // Skip Easy Apply jobs
				writer.Write([]string{job.Title, job.Link, "nan", "nan"}) // Ensure 4 columns
				count++
				if count >= 5 {
					break
				}
			}
		}

		fmt.Printf("✅ Stored %d jobs for %s (excluding Easy Apply)\n", count, title)
	}

	fmt.Println("📂 Job listings saved in Linkedin_jobs.csv")
	return nil
}


// Job Listings Handler
func JobListingsHandler(w http.ResponseWriter, r *http.Request) {
	// Set up chromedp with Chromium executable path (optional)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // Disable headless mode (optional)
		chromedp.Flag("executable-path", "/snap/bin/chromium"), // Uncomment and replace with actual path if needed
	)

	// Create a new context with the specified options
	allocatorCtx, allocatorCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocatorCancel() // Ensures Chrome instance cleanup

	// Create a new chromedp context using the allocator context
	ctx, ctxCancel := chromedp.NewContext(allocatorCtx)
	defer ctxCancel() // Ensures page cleanup

	jobTitles := []string{
		"Backend Developer",
	}
	location := "Berlin, Germany"
	dateSincePosted := ""

	// Fetch and store jobs
	if err := fetchAndStoreJobs(ctx, jobTitles, location, dateSincePosted); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching job listings: %v", err), http.StatusInternalServerError)
		return
	}

	// Clear browser session by resetting ChromeDP
	err := chromedp.Run(ctx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.Cancel(ctx) // Clears the session and resets context
		}),
	)

	if err != nil {
		fmt.Printf("❌ Failed to clear session: %v\n", err)
	}

	// Send a success message
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Job links saved in linkedin_jobs.csv"})
}



