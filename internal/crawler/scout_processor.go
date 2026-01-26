package crawler

import (
	"go-crawler/pkg/models"
)

type ScoutProcessor struct {
	Parser *Parser
	Filter URLFilter
}

func (s *ScoutProcessor) Process(url string) ([]models.URLQueue, []string, error) {
	// 1. FetchStatic and extract links
	allLinks, err := s.Parser.GetOutBoundLinks(url)
	if err != nil {
		return nil, nil, err
	}

	// 2. Filter logic (previously in scoutSite)
	var products []models.URLQueue
	var outbound []string

	for _, link := range allLinks {
		source := getDomain(link)

		// If it's a product, add to data results
		if s.Filter.Filter(source, link) {
			products = append(products, models.URLQueue{
				URL:    link,
				Domain: source.String(),
			})
		}
		// Always add to outbound to keep crawling
		outbound = append(outbound, link)
	}

	return products, outbound, nil
}
