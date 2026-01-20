package crawler

import (
	"go-crawler/pkg/models"
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
		if strings.Contains(link, "/p/") {
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
