package Linkedin

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	
	"encoding/csv"

	
	//"path/filepath"

	"github.com/chromedp/chromedp"
	
)


// LoadJobLinks loads job links from the CSV file and returns a map of job title -> links
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

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file is empty or contains only headers")
	}

	jobLinks := make(map[string][]string)

	for _, row := range records[1:] {
		if len(row) < 2 {
			continue
		}
		title, link := row[0], row[1]
		jobLinks[title] = append(jobLinks[title], link)
	}

	return jobLinks, nil
}

// StoreFailedJob stores a job that failed in a CSV file
func StoreFailedJob(title, link, reason string) error {
	file, err := os.OpenFile("storage/failed_jobs.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{title, link, reason, time.Now().Format(time.RFC3339)}); err != nil {
		return fmt.Errorf("❌ Failed to write to CSV: %v", err)
	}

	fmt.Printf("⚠️ Stored failed job: %s -> %s (Reason: %s)\n", title, link, reason)
	return nil
}

// StoreApplicationLink stores the application link in a CSV file
func StoreApplicationLink(title, link string) error {
	file, err := os.OpenFile("storage/Linkedin_joblinks.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("❌ Failed to open file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{title, link}); err != nil {
		return fmt.Errorf("❌ Failed to write to CSV: %v", err)
	}

	fmt.Printf("✅ Stored application: %s -> %s\n", title, link)
	return nil
}

// navigateAndClickApply navigates to the job page and attempts to click the apply button
func navigateAndClickApply(ctx context.Context, jobTitle, jobLink string) error {
	err := chromedp.Run(ctx,
		chromedp.Navigate(jobLink),
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		log.Printf("❌ Failed to navigate to job: %s -> %v\n", jobTitle, err)
		StoreFailedJob(jobTitle, jobLink, "Navigation failed")
		return err
	}

	err = chromedp.Run(ctx,
		chromedp.Click("div.jobs-apply-button--top-card button", chromedp.NodeVisible),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		log.Printf("⚠️ No apply button found for %s: %v\n", jobTitle, err)
		StoreFailedJob(jobTitle, jobLink, "Apply button missing")
		return err
	}

	return nil
}
