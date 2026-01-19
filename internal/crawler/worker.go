package crawler

import (
	"fmt"
	"go-crawler/internal"
	"go-crawler/pkg/models"
)

func CrawlWorker(
	id int,
	worklist chan []string,
	results chan<- models.PageData,
	visited *internal.SafeMap,
	domainMgr *DomainManager,
	parserService *Parser,
) {
	// Loop through the worklist channel
	for list := range worklist {
		for _, link := range list {
			// 1. Check if visited
			if visited.Contains(link) {
				continue
			}

			// 2. Polite Check (Robots.txt + Rate Limit)
			if !domainMgr.IsAllowed(link) {
				continue
			}
			domainMgr.Wait(link)

			// 3. Crawl
			fmt.Printf("[Worker %d] Crawling: %s\n", id, link)
			data, err := parserService.Parse(link)
			if err != nil {
				fmt.Printf("[Worker %d] Error: %v\n", id, err)
				continue
			}

			// 4. Send Data to DB Sink
			results <- data

			// 5. Add new links to queue
			// Run in a separate goroutine to avoid blocking the CrawlWorker if the queue is full
			fmt.Printf("Found %d links on %s. Queuing them...\n", len(data.OutboundLinks), link)
			go func(links []string) {
				worklist <- links
			}(data.OutboundLinks)
		}
	}
}
