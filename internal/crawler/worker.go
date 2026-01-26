package crawler

import (
	"fmt"
	"go-crawler/internal"
	"go-crawler/pkg/models"
	"strings"
)

func PageCrawlWorker(
	id int,
	worklist chan []string,
	results chan<- models.PageData,
	visited *internal.SafeMap,
	domainMgr *DomainManager,
	parserService *Parser,
) {

	gatherPageData := func(link string, filter URLFilter) ([]string, []models.PageData, error) {
		data, err := parserService.Parse(link)
		if err != nil {
			fmt.Printf("[Worker %d] Error: %v\n", id, err)
			return nil, nil, err
		}
		var returnList []models.PageData
		returnList = append(returnList, data)
		return data.OutboundLinks, returnList, nil
	}

	filter := AlwaysFilter{}
	StartCrawlWorker(id, worklist, results, visited, domainMgr, filter, gatherPageData)
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
	StartCrawlWorker(id, worklist, results, visited, domainMgr, filter, scout)
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
		for _, targetURL := range link {
			if targetURL == "" {
				continue
			}
			// Optional: Trim spaces just in case
			targetURL = strings.TrimSpace(targetURL)
			if targetURL == "" {
				continue
			}

			if visited.Contains(targetURL) {
				continue
			}
			if !domainMgr.IsAllowed(targetURL) {
				continue
			}
			domainMgr.Wait(targetURL)

			fmt.Printf("[Worker %d] Crawling: %s\n", id, targetURL)
			outBoundLinks, data, err := collectData(targetURL, filter)
			if err != nil {
				fmt.Printf("[Worker %d] Error: %v\n", id, err)
				continue
			}

			for _, entry := range data {
				results <- entry
			}

			fmt.Printf("Found %d links on %s. Queuing them...\n", len(outBoundLinks), targetURL)
			go func(links []string) {
				worklist <- links
			}(outBoundLinks)
		}
	}

}
