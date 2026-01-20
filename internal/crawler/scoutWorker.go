package crawler

import (
	"go-crawler/pkg/models"
)

type Crawler interface {
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

	for _, link := range outBoundLinks {
		source := getDomain(link)
		if filter.Filter(source, searchURL) {
			entry := models.URLQueue{
				URL:    link,
				Domain: source.String(),
			}
			productLinks = append(productLinks, entry)
		}
		outBoundLinks = append(outBoundLinks, link)
	}
	return outBoundLinks, productLinks, nil
}
