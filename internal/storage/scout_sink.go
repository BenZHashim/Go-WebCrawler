package storage

import (
	"database/sql"
	_ "github.com/jackc/pgx/v4/stdlib" // Import the driver
	"go-crawler/pkg/models"
	"log"
	"time"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

type ScoutingSink struct {
	*Storage
}

func (s *ScoutingSink) Save(batch []models.URLQueue) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO product_queue (url, domain, status) 
		VALUES ($1, $2, 'pending') 
		ON CONFLICT (url) DO NOTHING`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range batch {
		if _, err := stmt.Exec(item.URL, item.Domain); err != nil {
			log.Printf("Error inserting %s: %v", item.URL, err)
		}
	}

	return tx.Commit()
}

func startSaveWorker[T any](
	dataChan <-chan T,
	batchSize int,
	batchTimeout time.Duration,
	storage Storage,
	saveFunc func(Storage, []T) error) {

	buffer := make([]T, 0, batchSize)
	ticker := time.NewTicker(batchTimeout)
	defer ticker.Stop()

	flush := func() {
		if len(buffer) == 0 {
			return
		}

		err := saveFunc(storage, buffer)
		if err != nil {
			log.Printf("Batch save failed: %v", err)
			// In a real app, you might implement a retry mechanism here
		} else {
			log.Printf("Saved batch of size %d", len(buffer))
		}
		buffer = buffer[:0]
	}

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
			if len(buffer) >= batchSize {
				flush()
			}

		case <-ticker.C:
			// Trigger write if time is up
			flush()
		}
	}
}

func (storage Storage) StartPageWorker(dataChan <-chan models.PageData) {
	const (
		BatchSize    = 100
		BatchTimeout = 1 * time.Second
	)
	go startSaveWorker(dataChan, BatchSize, BatchTimeout, storage, savePageBatch)
}

func (storage Storage) StartProductQueueWorker(dataChan <-chan models.URLQueue) {

	saveToProductQueue := func(storage Storage, batch []models.URLQueue) error {
		tx, err := storage.db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		stmt, _ := tx.Prepare(`
                INSERT INTO product_queue (url, domain, status) 
                VALUES ($1, $2, 'pending') 
                ON CONFLICT (url) DO NOTHING`)

		defer stmt.Close()

		for _, url := range batch {
			stmt.Exec(url.URL, url.Domain)
		}
		return tx.Commit()
	}

	go startSaveWorker(dataChan, 10, 1*time.Second, storage, saveToProductQueue)
}

func savePageBatch(storage Storage, batch []models.PageData) error {
	tx, err := storage.db.Begin()
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
