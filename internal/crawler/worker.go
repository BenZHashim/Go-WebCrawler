package crawler

import (
	"fmt"
	"go-crawler/internal"
	"go-crawler/pkg/models"
)

func PageCrawlWorker(
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
			// Run in a separate goroutine to avoid blocking the PageCrawlWorker if the queue is full
			fmt.Printf("Found %d links on %s. Queuing them...\n", len(data.OutboundLinks), link)
			go func(links []string) {
				worklist <- links
			}(data.OutboundLinks)
		}
	}
}

func StartScoutCrawler(id int, worklist chan []string, results chan<- models.URLQueue, visited *internal.SafeMap, domainMgr *DomainManager, parserService *Parser) {
	filter := ProductFilter{}
	scout := func(link string, filter URLFilter) ([]string, []models.URLQueue, error) {
		entry, outboundLinks, err := scoutSite(link, *parserService, filter)
		if err != nil {
			return nil, nil, err
		}
		return entry, outboundLinks, err

	}
	go StartCrawlWorker(id, worklist, results, visited, domainMgr, filter, scout)
}

func StartCrawlWorker[T any](
	id int,
	worklist chan []string,
	results chan<- T,
	visited *internal.SafeMap,
	domainMgr *DomainManager,
	filter URLFilter,
	collectData func(link string, filter URLFilter) ([]string, []T, error),
) {

	for link := range worklist {
		for _, link := range link {
			if visited.Contains(link) {
				continue
			}
			if !domainMgr.IsAllowed(link) {
				continue
			}
			domainMgr.Wait(link)

			fmt.Printf("[Worker %d] Crawling: %s\n", id, link)
			outBoundLinks, data, err := collectData(link, filter)
			if err != nil {
				fmt.Printf("[Worker %d] Error: %v\n", id, err)
				continue
			}

			for _, entry := range data {
				results <- entry
			}

			fmt.Printf("Found %d links on %s. Queuing them...\n", len(outBoundLinks), link)
			go func(links []string) {
				worklist <- links
			}(outBoundLinks)
		}
	}

}
