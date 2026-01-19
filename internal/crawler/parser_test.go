package crawler

import (
	"strings"
	"testing"
)

func TestParser_Extract(t *testing.T) {

	p := &Parser{}
	baseURL := "https://example.com"

	rawHTML := `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Test Portfolio Page</title>
			<style>body { background: #000; }</style>
		</head>
		<body>
			<h1>Welcome to the Crawler</h1>
			<p>This is a <strong>test</strong> paragraph.</p>
			
			<div id="nav">
				<a href="/about">About Us</a>
				<a href="https://google.com">External Link</a>
			</div>

			<script>
				console.log("This text should NOT be extracted");
			</script>
		</body>
		</html>
	`
	reader := strings.NewReader(rawHTML)

	data, err := p.Extract(reader, baseURL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedTitle := "Test Portfolio Page"
	if data.Title != expectedTitle {
		t.Errorf("Title mismatch.\nExpected: %q\nGot: %q", expectedTitle, data.Title)
	}

	if strings.Contains(data.TextContent, "console.log") {
		t.Error("TextContent failed: Script content was not stripped out.")
	}
	if strings.Contains(data.TextContent, "body { background") {
		t.Error("TextContent failed: Style content was not stripped out.")
	}
	if !strings.Contains(data.TextContent, "Welcome to the Crawler") {
		t.Error("TextContent failed: Main H1 text missing.")
	}

	expectedLinks := []string{
		"https://example.com/about", // The relative link resolved
		"https://google.com",
	}

	if len(data.OutboundLinks) != len(expectedLinks) {
		t.Errorf("Link count mismatch. Expected %d, got %d", len(expectedLinks), len(data.OutboundLinks))
	} else {
		// Verify exact matches
		for i, link := range data.OutboundLinks {
			if link != expectedLinks[i] {
				t.Errorf("Link index %d mismatch.\nExpected: %s\nGot:      %s", i, expectedLinks[i], link)
			}
		}
	}
}
