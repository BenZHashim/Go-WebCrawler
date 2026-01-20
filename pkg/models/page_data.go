package models

import "time"

type PageData struct {
	URL           string
	Title         string
	TextContent   string
	StatusCode    int
	LoadTime      time.Duration
	OutboundLinks []string
}

type URLQueue struct {
	URL    string
	Domain string
}

type Product struct {
	Name        string
	Price       int
	ModelNumber string
	Source      string
}
