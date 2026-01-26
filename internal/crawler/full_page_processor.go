package crawler

import (
	"go-crawler/pkg/models"
)

// PageProcessor implements engine.Processor for scraping full page content.
type PageProcessor struct {
	Parser *Parser
}

// Process crawls a single page, extracting its text content and metadata.
func (p *PageProcessor) Process(url string) ([]models.PageData, []string, error) {
	// 1. Use your existing Parse method to get title, text, and links
	data, err := p.Parser.Parse(url)
	if err != nil {
		return nil, nil, err
	}

	// 2. Wrap the single PageData result into a slice
	results := []models.PageData{data}

	// 3. Return the data and the links to follow
	return results, data.OutboundLinks, nil
}
