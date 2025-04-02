package config

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// InitializeDatabase creates a SQLite database and a table if it doesn't exist
func InitializeDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "JSE.db")
	if err != nil {
		return nil, err
	}

	// Create jobs table
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS jobs (
		id TEXT PRIMARY KEY,
		jobid TEXT UNIQUE,  -- Unique job identifier
		title TEXT,
		company TEXT,
		location TEXT,
		posted_date TEXT,
		link TEXT UNIQUE,  -- Ensure the link is unique
		processed BOOLEAN
	);`


	_, err = db.Exec(createTableQuery)
	if err != nil {
		return nil, err
	}

	fmt.Println("✅ Database initialized")
	return db, nil
}
