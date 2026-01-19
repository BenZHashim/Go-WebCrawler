package main

import (
	"database/sql"
	_ "github.com/jackc/pgx/v4/stdlib" // Import the driver
	"go-crawler/models"
	"log"
	"time"
)

// storageWorker listens to a channel and writes data to the DB
func batchedStorageWorker(db *sql.DB, dataChan <-chan models.PageData) {
	const (
		BatchSize    = 100             // Write when we have this many
		BatchTimeout = 1 * time.Second // Or write when this much time passes
	)

	buffer := make([]models.PageData, 0, BatchSize)
	ticker := time.NewTicker(BatchTimeout)
	defer ticker.Stop()

	// Define the save logic as a helper function to avoid code duplication
	flush := func() {
		if len(buffer) == 0 {
			return
		}

		err := saveBatch(db, buffer)
		if err != nil {
			log.Printf("Batch save failed: %v", err)
			// In a real app, you might implement a retry mechanism here
		} else {
			log.Printf("Saved batch of %d pages", len(buffer))
		}

		// Reset the buffer (keep capacity to avoid reallocation)
		buffer = buffer[:0]
	}

	// The Main Event Loop
	for {
		select {
		case data, ok := <-dataChan:
			// If the channel is closed by main, flush what we have and exit
			if !ok {
				flush()
				return
			}

			// Add to buffer
			buffer = append(buffer, data)

			// Trigger write if buffer is full
			if len(buffer) >= BatchSize {
				flush()
			}

		case <-ticker.C:
			// Trigger write if time is up
			flush()
		}
	}
}

func saveBatch(db *sql.DB, batch []models.PageData) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare statements
	stmtPage, _ := tx.Prepare(`
		INSERT INTO pages (url, title, content_text, status_code, load_time_ms, crawled_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (url) DO NOTHING`)

	defer stmtPage.Close()

	for _, p := range batch {
		_, err := stmtPage.Exec(p.URL, p.Title, p.TextContent, p.StatusCode, p.LoadTime.Milliseconds(), time.Now())
		if err != nil {
			continue
		}
	}
	return tx.Commit()
}
