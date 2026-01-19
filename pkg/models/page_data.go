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
