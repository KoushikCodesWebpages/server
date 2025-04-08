package Linkedin

import (
	"context"
	"fmt"
	"log"
	"time"
	
	"database/sql"

	
	//"path/filepath"


	"github.com/chromedp/chromedp"
)

/// JobTemp struct for temporary processing (matches database columns)
type JobTemp struct {
	ID    string `json:"id"`    // Matches id TEXT in linkedin_jobs
	Title string `json:"title"` // Matches title TEXT
	Link  string `json:"link"`  // Matches link TEXT
}

// LoadJobLinksFromDB fetches job links from linkedin_jobs
func LoadJobLinksFromDB(db *sql.DB) (map[string][]JobTemp, error) {
	rows, err := db.Query("SELECT id, title, link FROM linkedin_jobs")
	if err != nil {
		return nil, fmt.Errorf("failed to load job links: %w", err)
	}
	defer rows.Close()

	jobLinks := make(map[string][]JobTemp)

	for rows.Next() {
		var job JobTemp
		if err := rows.Scan(&job.ID, &job.Title, &job.Link); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		jobLinks[job.Title] = append(jobLinks[job.Title], job)
	}

	return jobLinks, nil
}

type Joblinks struct {
	ID    int    `json:"id"`    // Matches INTEGER PRIMARY KEY AUTOINCREMENT
	JobID string `json:"job_id"` // Matches job_id TEXT (foreign key)
	Link  string `json:"link"`   // Matches job_link TEXT
}

// StoreApplicationLink stores the application link in the database
func StoreApplicationLink(db *sql.DB, jobID, link string) error {
	_, err := db.Exec(`
        INSERT INTO linkedin_job_application_links (job_id, job_link) 
        VALUES (?, ?)`, jobID, link, // Fixed incorrect table & column name
	)
	if err != nil {
		log.Printf("❌ Failed to store application link in DB: %v\n", err)
		return err
	}

	fmt.Printf("✅ Stored application link: %s\n", link)
	return nil
}

type FailedJob struct {
	ID      int    `json:"id"`      // Matches INTEGER PRIMARY KEY AUTOINCREMENT
	JobID   string `json:"job_id"`  // Matches job_id TEXT
	JobLink string `json:"job_link"` // Matches job_link TEXT
}

// StoreFailedJob stores a failed job in the database
func StoreFailedJob(db *sql.DB, jobID, jobLink, reason string) error {
	_, err := db.Exec(`
        INSERT INTO linkedin_failed_jobs (job_id, job_link, reason) 
        VALUES (?, ?, ?)`, jobID, jobLink, reason,
	)
	if err != nil {
		log.Printf("❌ Failed to store failed job in DB: %v\n", err)
		return err
	}

	fmt.Printf("⚠️ Stored failed job: %s -> %s (Reason: %s)\n", jobID, jobLink, reason)
	return nil
}

// navigateAndClickApply navigates to the job page and attempts to click the apply button
func navigateAndClickApply(ctx context.Context, db *sql.DB, jobID string, jobLink string) error {
	// Navigate to the job posting
	err := chromedp.Run(ctx,
		chromedp.Navigate(jobLink),
		chromedp.Sleep(5*time.Second),
	)
	if err != nil {
		log.Printf("❌ Failed to navigate to job: %s -> %v\n", jobID, err)
		StoreFailedJob(db, jobID, jobLink, "Navigation failed")
		return err
	}

	// More targeted selector using parent div and apply button
	err = chromedp.Run(ctx,
		chromedp.Click(`div.main-actions__ActionsContainer-sc-68c89ebb-0 button[data-testid="apply-button"]`, chromedp.NodeVisible),
		chromedp.Sleep(3*time.Second),
	)
	if err != nil {
		log.Printf("⚠️ Apply button not found for jobID %s: %v\n", jobID, err)
		StoreFailedJob(db, jobID, jobLink, "Apply button missing")
		return err
	}

	return nil
}


