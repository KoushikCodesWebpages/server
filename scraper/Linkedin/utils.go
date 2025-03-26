package Linkedin

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	
	"encoding/csv"

	
	//"path/filepath"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/target"
	
)

func StopChrome() {
	if chromeCmd != nil {
		fmt.Println("🛑 Closing Chromium...")
		if err := chromeCmd.Process.Kill(); err != nil {
			fmt.Printf("⚠️ Failed to close Chromium: %v\n", err)
		} else {
			fmt.Println("✅ Chromium closed successfully.")
		}
	}
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
		chromedp.Click(`div.jobs-apply-button--top-card button`, chromedp.NodeVisible),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		log.Printf("⚠️ No apply button found for %s: %v\n", jobTitle, err)
		StoreFailedJob(jobTitle, jobLink, "Apply button missing")
		return err
	}

	return nil
}

// captureAndCloseNewTab captures the application tab and closes it
func captureAndCloseNewTab(ctx context.Context, jobTitle string, existingTabs map[target.ID]struct{}) ([]string, error) {
	var capturedURLs []string
	var newTabID target.ID

	newTabs, err := chromedp.Targets(ctx)
	if err != nil {
		log.Printf("❌ Failed to get updated open tabs: %v\n", err)
		return nil, err
	}

	// Identify new non-LinkedIn tabs
	for _, t := range newTabs {
		if _, exists := existingTabs[t.TargetID]; !exists && t.Type == "page" && t.URL != "" && !strings.Contains(t.URL, "linkedin.com") {
			capturedURLs = append(capturedURLs, t.URL)

			if err := StoreApplicationLink(jobTitle, t.URL); err != nil {
				log.Printf("❌ Error saving URL: %v\n", err)
			}

			fmt.Println("✅ Captured application page:", t.URL)
			newTabID = t.TargetID
			break
		}
	}

	// Close the new tab if it was opened
	if newTabID != "" {
		tabCtx, cancel := chromedp.NewContext(ctx, chromedp.WithTargetID(newTabID))
		defer cancel()

		err := chromedp.Run(tabCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			return target.CloseTarget(newTabID).Do(ctx)
		}))
		if err != nil {
			log.Printf("❌ Error closing tab: %v\n", err)
		} else {
			fmt.Println("✅ Successfully closed extra tab:", newTabID)
		}
	}

	return capturedURLs, nil
}


func InitializeCSVFiles() error {
	// Define headers
	failedJobsHeaders := []string{"Job Title", "Job Link", "Reason", "Timestamp"}
	applicationLinksHeaders := []string{"Job Title", "Job Link"}

	// Initialize files with headers if they are empty
	if err := createCSVWithHeaders("storage/failed_jobs.csv", failedJobsHeaders); err != nil {
		return err
	}
	if err := createCSVWithHeaders("storage/Linkedin_joblinks.csv", applicationLinksHeaders); err != nil {
		return err
	}

	return nil
}

func createCSVWithHeaders(filePath string, headers []string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath) // Creates a new file
		if err != nil {
			return fmt.Errorf("❌ Failed to create file %s: %v", filePath, err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		// Write headers
		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("❌ Failed to write headers: %v", err)
		}
		fmt.Printf("✅ Created CSV file with headers: %s\n", filePath)
	}
	return nil
}
