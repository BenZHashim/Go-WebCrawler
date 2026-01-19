package main

import (
	"github.com/temoto/robotstxt"
	"net/http"
	"net/url"
)

// CheckRobots returns true if the userAgent is allowed to visit the specific path
func robotAllowed(targetURL, userAgent string) bool {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return false
	}

	// Construct the root URL to find robots.txt (e.g., https://example.com/robots.txt)
	robotsURL := parsedURL.Scheme + "://" + parsedURL.Host + "/robots.txt"

	resp, err := http.Get(robotsURL)
	if err != nil {
		// If robots.txt doesn't exist, it usually means crawl whatever you want.
		// However, for safety, some crawlers default to false here.
		return true
	}
	defer resp.Body.Close()

	data, err := robotstxt.FromResponse(resp)
	if err != nil {
		return true
	}

	return data.TestAgent(targetURL, userAgent)
}
