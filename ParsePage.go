package main

import (
	"fmt"
	"go-crawler/models"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// parsePage fetches the page and returns all URLs found in href tags
func parsePage(targetURL string) (models.PageData, error) {

	start := time.Now()
	pageData := models.PageData{URL: targetURL}
	// Define a custom client with timeout
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return pageData, err
	}

	// Identify yourself! (Use your own name/email so they know you are learning)
	req.Header.Set("User-Agent", "MyLearningCrawler/1.0 (benjaminzhashim@gmail.com)")

	resp, err := client.Do(req)
	defer resp.Body.Close()

	fmt.Printf("DEBUG: Fetched %s - Status: %d\n", targetURL, resp.StatusCode)

	pageData.LoadTime = time.Since(start)

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return pageData, err
	}

	var links []string
	var title string
	var textBuilder strings.Builder

	// Recursive generic HTML walker
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			title = n.FirstChild.Data
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					// Resolve relative URLs
					absoluteURL := resolveURL(targetURL, a.Val)
					if absoluteURL != "" {
						links = append(links, absoluteURL)
					}
				}
			}
		}
		if n.Type == html.TextNode {
			parent := n.Parent
			if parent != nil && parent.Data != "script" && parent.Data != "style" {
				text := strings.TrimSpace(n.Data)
				if len(text) > 0 {
					textBuilder.WriteString(text + " ")
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	pageData.Title = title
	pageData.TextContent = textBuilder.String()
	pageData.StatusCode = resp.StatusCode
	pageData.OutboundLinks = links
	return pageData, nil
}

// Utility to resolve relative URLs (e.g. "/about" -> "https://site.com/about")
func resolveURL(base, href string) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	return baseURL.ResolveReference(u).String()
}
