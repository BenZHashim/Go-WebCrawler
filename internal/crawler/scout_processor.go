package crawler

import (
	"go-crawler/pkg/models"
	"regexp"
	"strings"
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
			link := getProductURL(link)
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

func scoutSite(searchURL string,
	parser Parser,
	filter URLFilter,
) ([]string, []models.URLQueue, error) {

	var outBoundLinks []string
	var productLinks []models.URLQueue

	outBoundLinks, err := parser.GetOutBoundLinks(searchURL)
	if err != nil {
		return nil, nil, err
	}
	var goToLinks []string
	for _, link := range outBoundLinks {
		source := getDomain(link)
		if filter.Filter(source, link) {
			link := getProductURL(link)
			entry := models.URLQueue{
				URL:    link,
				Domain: source.String(),
			}
			productLinks = append(productLinks, entry)
			goToLinks = append(goToLinks, link)
		}

	}
	return goToLinks, productLinks, nil
}

// Main configuration for the Scout
var siteRules = map[string]*regexp.Regexp{
	"amazon.com":  regexp.MustCompile(`\/dp\/([A-Z0-9]{10})`),
	"newegg.com":  regexp.MustCompile(`\/p\/([A-Z0-9]+)`),
	"bestbuy.com": regexp.MustCompile(`(\d{7})\.p`),
}

func getProductURL(link string) string {
	// Loop through our rules
	for domain, regex := range siteRules {
		if strings.Contains(link, domain) {
			matches := regex.FindStringSubmatch(link)
			if len(matches) > 1 {
				// Return the cleaned URL and the domain name
				// e.g. "https://amazon.com/dp/B01...", "amazon.com"
				return cleanUpURL(domain, matches[1])
			}
		}
	}
	return ""
}

func cleanUpURL(domain string, id string) string {
	switch {
	case strings.Contains(domain, "amazon.com"):
		return "https://www.amazon.com/dp/" + id

	case strings.Contains(domain, "newegg.com"):
		return "https://www.newegg.com/p/" + id

	case strings.Contains(domain, "bestbuy.com"):
		return "https://www.bestbuy.com/site/" + id + ".p"

	default:
		return ""
	}
}
