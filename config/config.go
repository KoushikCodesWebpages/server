package config

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// InitializeDatabase creates the SQLite database with LinkedIn and Xing tables
func InitializeDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "JSE.db")
	if err != nil {
		return nil, err
	}

	// Create LinkedIn jobs table
	createLinkedInJobsTable := `
	CREATE TABLE IF NOT EXISTS linkedin_jobs (
		id TEXT PRIMARY KEY,
		jobid TEXT UNIQUE,  
		title TEXT,
		company TEXT,
		location TEXT,
		posted_date TEXT,
		link TEXT UNIQUE,  
		processed BOOLEAN
	);`

	// Create Xing jobs table
	createXingJobsTable := `
	CREATE TABLE IF NOT EXISTS xing_jobs (
		id TEXT PRIMARY KEY,
		jobid TEXT UNIQUE,  		
		title TEXT,
		company TEXT,
		location TEXT,
		posted_date TEXT,
		link TEXT UNIQUE,  
		processed BOOLEAN
	);`

	// Create LinkedIn failed jobs table
	createLinkedInFailedJobsTable := `
	CREATE TABLE IF NOT EXISTS linkedin_failed_jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT,
		job_link TEXT UNIQUE,
		FOREIGN KEY (job_id) REFERENCES linkedin_jobs(id) ON DELETE CASCADE
	);`

	// Create Xing failed jobs table
	createXingFailedJobsTable := `
	CREATE TABLE IF NOT EXISTS xing_failed_jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT,
		job_link TEXT UNIQUE,
		FOREIGN KEY (job_id) REFERENCES xing_jobs(id) ON DELETE CASCADE
	);`

	// Create LinkedIn job application links table
	createLinkedInJobApplicationLinksTable := `
	CREATE TABLE IF NOT EXISTS linkedin_job_application_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT,
		job_link TEXT UNIQUE,
		FOREIGN KEY (job_id) REFERENCES linkedin_jobs(id) ON DELETE CASCADE
	);`

	// Create Xing job application links table
	createXingJobApplicationLinksTable := `
	CREATE TABLE IF NOT EXISTS xing_job_application_links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id TEXT,
		job_link TEXT,
		UNIQUE(job_id, job_link),  -- ✅ Prevent duplicates
		FOREIGN KEY (job_id) REFERENCES xing_jobs(id) ON DELETE CASCADE
	);`
	

	// Execute table creation queries
	for _, query := range []string{
		createLinkedInJobsTable,
		createXingJobsTable,
		createLinkedInFailedJobsTable,
		createXingFailedJobsTable,
		createLinkedInJobApplicationLinksTable,
		createXingJobApplicationLinksTable,
	} {
		if _, err = db.Exec(query); err != nil {
			return nil, fmt.Errorf("❌ Failed to create table: %v", err)
		}
	}

	fmt.Println("✅ Database initialized successfully with separate LinkedIn & Xing tables")
	return db, nil
}
