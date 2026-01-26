package engine

import (
	"context"
	"fmt"
	"go-crawler/internal"
	"go-crawler/internal/crawler"
	"log"
	"sync"
	"time"
)

// Processor defines how to crawl a single page.
// It returns extracted data items (T) and new links to follow.
type Processor[T any] interface {
	Process(url string) (data []T, links []string, err error)
}

// Sink defines how to persist the data.
type Sink[T any] interface {
	Save(batch []T) error
}

// Config holds worker settings.
type Config struct {
	Workers   int
	BatchSize int
	RateLimit time.Duration
}

// Engine orchestrates the crawling process.
type Engine[T any] struct {
	config    Config
	processor Processor[T]
	sink      Sink[T]

	// State
	visited   *internal.SafeMap
	domainMgr *crawler.DomainManager
	worklist  chan []string
	results   chan T
	waitGroup sync.WaitGroup
}

func NewEngine[T any](cfg Config, proc Processor[T], sink Sink[T], domainMgr *crawler.DomainManager) *Engine[T] {
	return &Engine[T]{
		config:    cfg,
		processor: proc,
		sink:      sink,
		visited:   internal.NewSafeMap(),
		domainMgr: domainMgr,
		worklist:  make(chan []string, 100),
		results:   make(chan T, cfg.BatchSize*2),
	}
}

// Run starts the crawler and blocks until context is cancelled or manual stop.
func (engine *Engine[T]) Run(ctx context.Context, startURLs ...string) {
	// 1. Start Storage Worker
	engine.waitGroup.Add(1)
	go engine.startStorageWorker(ctx)

	// 2. Start Crawler Workers
	for i := 0; i < engine.config.Workers; i++ {
		engine.waitGroup.Add(1)
		go engine.startCrawlWorker(ctx, i)
	}

	// 3. Seed the worklist
	go func() {
		engine.worklist <- startURLs
	}()

	fmt.Printf("Engine started with %d workers\n", engine.config.Workers)
	engine.waitGroup.Wait()
}

func (engine *Engine[T]) startCrawlWorker(ctx context.Context, id int) {
	defer engine.waitGroup.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case list := <-engine.worklist:
			for _, link := range list {
				// Checks & Rate Limiting handled by the Engine, not the Processor
				if engine.visited.Contains(link) || !engine.domainMgr.IsAllowed(link) {
					continue
				}
				engine.domainMgr.Wait(link)

				fmt.Printf("[Worker %d] Processing: %s\n", id, link)

				// Execute the Strategy
				data, outbound, err := engine.processor.Process(link)
				if err != nil {
					log.Printf("[Worker %d] Error: %v", id, err)
					continue
				}

				// Send results to storage
				for _, item := range data {
					engine.results <- item
				}

				// Queue new links
				// (Non-blocking send optimization could go here)
				go func(l []string) { engine.worklist <- l }(outbound)
			}
		}
	}
}

func (engine *Engine[T]) startStorageWorker(ctx context.Context) {
	defer engine.waitGroup.Done()
	buffer := make([]T, 0, engine.config.BatchSize)
	ticker := time.NewTicker(2 * time.Second) // Flush interval
	defer ticker.Stop()

	flush := func() {
		if len(buffer) == 0 {
			return
		}
		if err := engine.sink.Save(buffer); err != nil {
			log.Printf("Failed to save batch: %v", err)
		} else {
			log.Printf("Saved batch of %d items", len(buffer))
		}
		buffer = buffer[:0] // Reset buffer
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case item := <-engine.results:
			buffer = append(buffer, item)
			if len(buffer) >= engine.config.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}
