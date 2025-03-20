package scraper

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
	Title string `json:"title"`
	Link  string `json:"link"`
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
	file, err := os.Create("storage/job_links.csv")
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	writer.Write([]string{"Title", "Job Link"})

	for _, title := range jobTitles {
		searchURL := constructSearchUrl(title, location, dateSincePosted)
		var jobs []Job

		err := chromedp.Run(ctx,
			chromedp.Navigate(searchURL),
			chromedp.Sleep(5*time.Second),
			chromedp.WaitVisible(`.jobs-search__results-list`, chromedp.ByQuery),
			chromedp.Evaluate(`Array.from(document.querySelectorAll('.jobs-search__results-list li')).map(el => ({
				title: "`+title+`",
				link: el.querySelector('.base-card__full-link')?.href || ''
			}))`, &jobs),
		)

		if err != nil {
			fmt.Printf("❌ Failed to fetch jobs for %s: %v\n", title, err)
			continue
		}

		// Limit to 5 job links per title
		count := 0
		for _, job := range jobs {
			if job.Link != "" {
				writer.Write([]string{job.Title, job.Link})
				count++
				if count >= 5 {
					break
				}
			}
		}

		fmt.Printf("✅ Stored %d jobs for %s\n", count, title)
	}

	fmt.Println("📂 Job listings saved in job_links.csv")
	return nil
}

// Job Listings Handler
func JobListingsHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	jobTitles := []string{
		"Software Engineer", "Data Scientist", "Product Manager",
		"DevOps Engineer", "Cybersecurity Analyst", "Cloud Engineer",
		"Machine Learning Engineer", "Frontend Developer", "Backend Developer", "QA Engineer",
	}
	location := "Berlin, Germany"
	dateSincePosted := ""

	if err := fetchAndStoreJobs(ctx, jobTitles, location, dateSincePosted); err != nil {
		http.Error(w, fmt.Sprintf("Error fetching job listings: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Job links saved in job_links.csv"})
}
