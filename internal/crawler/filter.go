package crawler

import (
	"fmt"
	"go-crawler/pkg/models"
	"net/url"
	"strings"
)

type URLFilter interface {
	Filter(source models.DataSource, link string) bool
}

type ProductFilter struct{}

func (filter ProductFilter) Filter(source models.DataSource, link string) bool {
	switch source {
	case models.Amazon:
		if strings.Contains(link, "/dp/") {
			return true
		}
	case models.Newegg:
		if strings.Contains(link, "/p/") && strings.Contains(link, "corsair") {
			return true
		}
	default:
		return false
	}
	return false
}

type AlwaysFilter struct{}

func (filter AlwaysFilter) Filter(source models.DataSource, link string) bool {
	return true
}

type InDomainFilter struct {
	Domain string
}

func NewInDomainFilter(startURL string) (*InDomainFilter, error) {
	u, err := url.Parse(startURL)
	if err != nil {
		return nil, fmt.Errorf("invalid start URL: %w", err)
	}

	// Extract hostname and strip "www." to allow subdomains
	host := u.Hostname()
	domain := strings.TrimPrefix(host, "www.")

	if domain == "" {
		return nil, fmt.Errorf("could not extract domain from %s", startURL)
	}

	return &InDomainFilter{Domain: domain}, nil
}

func (filter InDomainFilter) Filter(source models.DataSource, link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	return strings.Contains(strings.ToLower(u.Host), strings.ToLower(filter.Domain))
}
