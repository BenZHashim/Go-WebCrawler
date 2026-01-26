package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"go-crawler/internal/crawler"
	"go-crawler/internal/crawler/engine"
	"go-crawler/internal/storage"
	"go-crawler/pkg/models"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. Setup
	startURL := flag.String("url", "https://www.hollywoodreporter.com", "Starting URL")
	workers := flag.Int("workers", 10, "Worker count")
	flag.Parse()

	log.Printf("Starting crawler with URL %s", *startURL)

	db := waitForDB(os.Getenv("DB_URL"))
	defer db.Close()

	store := storage.NewStorage(db)
	parser := crawler.NewParser("MyPageCrawler/1.0")
	domainMgr := crawler.NewDomainManager(2 * time.Second)

	filter, err := crawler.NewInDomainFilter(*startURL)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Define Strategies for Page Content
	// Strategy: Parse full content
	pageProc := &crawler.PageProcessor{
		Parser: parser,
		Filter: filter,
	}
	// Sink: Save to 'pages' table
	pageSink := &storage.PageSink{Storage: store}

	// 3. Initialize Engine with [models.PageData]
	// Note: We increase BatchSize because page data is larger than product links
	crawlerEngine := engine.NewEngine[models.PageData](
		engine.Config{Workers: *workers, BatchSize: 20},
		pageProc,
		pageSink,
		domainMgr,
	)

	// 4. Run
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
		<-stopChan
		cancel()
	}()

	log.Println("Starting Page Content Crawler...")
	crawlerEngine.Run(ctx, *startURL)
}

func waitForDB(url string) *sql.DB {
	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("pgx", url)
		if err == nil {
			if err = db.Ping(); err == nil {
				fmt.Println("Connected to migrations!")
				return db
			}
		}
		fmt.Printf("Waiting for DB... (%v)\n", err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("Could not connect to DB after retries")
	return nil
}
