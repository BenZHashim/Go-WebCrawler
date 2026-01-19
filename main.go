package main

import (
	"database/sql"
	"fmt"
	"github.com/temoto/robotstxt"
	"go-crawler/models"
	"golang.org/x/time/rate"
	"log"
	"os"
	"time"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {

	numWorkers := 10
	startURL := "https://en.wikipedia.org/wiki/Main_Page" // Change this to your target

	dbURL := os.Getenv("DB_URL")

	db := waitForDB(dbURL)
	defer db.Close()

	// 3. Initialize Channels and State
	worklist := make(chan []string)            // Queue of links to process
	results := make(chan models.PageData, 100) // Data ready to be saved
	visited := SafeMap{v: make(map[string]bool)}

	domainManager := &DomainManager{
		limiters:    make(map[string]*rate.Limiter),
		robotsCache: make(map[string]*robotstxt.Group),
	}

	go batchedStorageWorker(db, results)

	for i := 0; i < numWorkers; i++ {
		// We just call the named function now
		go worker(i, worklist, results, &visited, domainManager)
	}

	fmt.Println("Starting Crawler...")
	go func() { worklist <- []string{startURL} }()

	select {}
}

func waitForDB(url string) *sql.DB {
	var db *sql.DB
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("pgx", url)
		if err == nil {
			if err = db.Ping(); err == nil {
				fmt.Println("Connected to Database!")
				return db
			}
		}
		fmt.Printf("Waiting for DB... (%v)\n", err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("Could not connect to DB after retries")
	return nil
}
