package Linkedin

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/target"
)

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

// captureAndCloseNewTab captures the application link and stores it in linkedin_job_application_links
func captureAndCloseNewTab(ctx context.Context, db *sql.DB, jobID string, existingTabs map[target.ID]struct{}) ([]string, error) {
	var capturedURLs []string
	var newTabID target.ID

	// Get all open tabs
	newTabs, err := chromedp.Targets(ctx)
	if err != nil {
		log.Printf("❌ Failed to get updated open tabs: %v\n", err)
		return nil, err
	}

	// Find new tab that is NOT a LinkedIn page
	for _, t := range newTabs {
		if _, exists := existingTabs[t.TargetID]; !exists && t.Type == "page" && t.URL != "" && !strings.Contains(t.URL, "linkedin.com") {
			cleanURL := strings.TrimSpace(t.URL)
			if cleanURL == "" {
				continue
			}

			capturedURLs = append(capturedURLs, cleanURL)

			// ✅ Store captured application link in linkedin_job_application_links
			if err := StoreApplicationLink(db, jobID, cleanURL); err != nil {
				log.Printf("❌ Error storing application link in DB: %v\n", err)
			} else {
				fmt.Println("✅ Captured and stored application page:", cleanURL)
			}

			newTabID = t.TargetID
			break // Stop after capturing one application link
		}
	}

	// Close the new tab if found
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