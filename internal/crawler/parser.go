package crawler

import (
	"go-crawler/pkg/models"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Parser struct {
	UserAgent string
}

func NewParser(userAgent string) *Parser {
	return &Parser{UserAgent: userAgent}
}

func (p *Parser) Parse(targetURL string) (models.PageData, error) {
	// 1. Fetch the raw stream
	start := time.Now()
	body, statusCode, err := p.Fetch(targetURL)
	loadTime := time.Since(start)

	if err != nil {
		return models.PageData{URL: targetURL}, err
	}
	defer body.Close()

	// 2. Extract data from the stream
	// Note: We pass the URL separately to resolve relative links (e.g. "/about")
	data, err := p.Extract(body, targetURL)
	if err != nil {
		return models.PageData{URL: targetURL, StatusCode: statusCode}, err
	}

	// 3. Enrich with metadata
	data.LoadTime = loadTime
	data.StatusCode = statusCode

	return data, nil
}

func (p *Parser) Fetch(targetURL string) (io.ReadCloser, int, error) {
	client := http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("User-Agent", p.UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}

	return resp.Body, resp.StatusCode, nil
}

func (p *Parser) Extract(r io.Reader, baseURL string) (models.PageData, error) {
	data := models.PageData{URL: baseURL}

	doc, err := html.Parse(r)
	if err != nil {
		return data, err
	}

	var links []string
	var textBuilder strings.Builder

	var visit func(*html.Node)
	visit = func(n *html.Node) {
		// 1. Find Title
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			data.Title = n.FirstChild.Data
		}

		// 2. Find Links
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					absoluteURL := resolveURL(baseURL, a.Val)
					if absoluteURL != "" {
						links = append(links, absoluteURL)
					}
				}
			}
		}

		// 3. Extract Text (ignoring scripts/styles)
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

	data.TextContent = textBuilder.String()
	data.OutboundLinks = links
	return data, nil
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
