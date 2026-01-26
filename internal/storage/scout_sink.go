package storage

import (
	"database/sql"
	_ "github.com/jackc/pgx/v4/stdlib" // Import the driver
	"go-crawler/pkg/models"
	"log"
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
