package Linkedin

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	
	"encoding/csv"

	
	//"path/filepath"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/target"
	
)

// InitializeCSVFiles initializes the required CSV files with headers
func InitializeCSVFiles() error {
	failedJobsHeaders := []string{"Job Title", "Job Link", "Reason", "Timestamp"}
	applicationLinksHeaders := []string{"Job Title", "Company", "Description", "Job Link"}

	if err := createCSVWithHeaders("storage/failed_jobs.csv", failedJobsHeaders); err != nil {
		return err
	}
	if err := createCSVWithHeaders("storage/Linkedin_joblinks.csv", applicationLinksHeaders); err != nil {
		return err
	}

	return nil
}


// createCSVWithHeaders creates a CSV file with headers if it doesn't exist
func createCSVWithHeaders(filePath string, headers []string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("❌ Failed to create file %s: %v", filePath, err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		if err := writer.Write(headers); err != nil {
			return fmt.Errorf("❌ Failed to write headers: %v", err)
		}
		fmt.Printf("✅ Created CSV file with headers: %s\n", filePath)
	}
	return nil
}


// StopChrome closes the Chromium browser process
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


// captureAndCloseNewTab captures the application tab and closes it
func captureAndCloseNewTab(ctx context.Context, jobTitle string, existingTabs map[target.ID]struct{}) ([]string, error) {
	var capturedURLs []string
	var newTabID target.ID

	newTabs, err := chromedp.Targets(ctx)
	if err != nil {
		log.Printf("❌ Failed to get updated open tabs: %v\n", err)
		return nil, err
	}

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
