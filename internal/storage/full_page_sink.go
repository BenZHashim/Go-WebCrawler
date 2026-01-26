package storage

import (
	"go-crawler/pkg/models"
	"log"
	"time"
)

// PageSink implements engine.Sink for saving full page content to Postgres.
type PageSink struct {
	*Storage
}

func (s *PageSink) Save(batch []models.PageData) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare statement (adapted from your original internal/storage/storage.go)
	stmt, err := tx.Prepare(`
		INSERT INTO pages (url, title, content_text, status_code, load_time_ms, crawled_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (url) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, p := range batch {
		// Log or track specific errors if needed, otherwise continue
		_, err := stmt.Exec(
			p.URL,
			p.Title,
			p.TextContent,
			p.StatusCode,
			p.LoadTime.Milliseconds(),
			time.Now(),
		)
		if err != nil {
			log.Printf("Error saving page %s: %v", p.URL, err)
		}
	}

	return tx.Commit()
}
