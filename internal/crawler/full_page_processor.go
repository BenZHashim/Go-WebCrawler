package crawler

import (
	"go-crawler/pkg/models"
)

// PageProcessor implements engine.Processor for scraping full page content.
type PageProcessor struct {
	Parser *Parser
	Filter URLFilter
}

// Process crawls a single page, extracting its text content and metadata.
func (processor *PageProcessor) Process(url string) ([]models.PageData, []string, error) {
	// 1. Use your existing Parse method to get title, text, and links
	data, err := processor.Parser.Parse(url)
	if err != nil {
		return nil, nil, err
	}

	var validLinks []string
	for _, link := range data.OutboundLinks {
		// We pass 'models.None' as the source because InDomainFilter ignores it anyway
		if processor.Filter.Filter(models.None, link) {
			validLinks = append(validLinks, link)
		}
	}

	// 2. Wrap the single PageData result into a slice
	results := []models.PageData{data}

	// 3. Return the data and the links to follow
	return results, validLinks, nil
}
