package main

import (
	"database/sql"
	"flag"
	"fmt"
	"go-crawler/internal"
	"go-crawler/internal/crawler"
	"go-crawler/internal/storage"
	"go-crawler/pkg/models"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {

	defaultURL := "https://www.newegg.com/p/pl?d=corsair"
	defaultWorkers := 10

	// 2. Parse Flags (allows running: ./crawler -url="https://google.com" -workers=50)
	startURL := flag.String("url", defaultURL, "The starting URL to crawl")
	numWorkers := flag.Int("workers", defaultWorkers, "Number of concurrent workers")
	flag.Parse()

	dbURL := os.Getenv("DB_URL")

	db := waitForDB(dbURL)
	defer db.Close()

	// 3. Initialize Channels and State
	worklist := make(chan []string) // Queue of links to process
	//pageResults := make(chan models.PageData, 100) // Data ready to be saved
	queueResults := make(chan models.URLQueue, 100)
	visited := internal.NewSafeMap()

	domainManager := crawler.NewDomainManager()

	storageWorker := storage.NewStorage(db)
	//go storageWorker.StartPageWorker(pageResults)
	go storageWorker.StartProductQueueWorker(queueResults)

	parserService := crawler.NewParser("MyPortfolioCrawler/1.0 (benjaminzhashim@gmail.com)")

	for i := 0; i < *numWorkers; i++ {
		// We just call the named function now
		//go crawler.PageCrawlWorker(i, worklist, pageResults, visited, domainManager, parserService)
		go crawler.StartScoutCrawler(i, worklist, queueResults, visited, domainManager, parserService)
	}

	fmt.Println("Starting Crawler...")
	go func() { worklist <- []string{*startURL} }()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Block here until a signal is received
	<-stopChan
	log.Println("Shutting down")

	db.Close()
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
