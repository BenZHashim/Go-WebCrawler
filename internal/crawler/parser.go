package crawler

import (
	"bytes"
	"fmt"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"go-crawler/pkg/models"
	"golang.org/x/net/context"
	"golang.org/x/net/html"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type FetchAction int

const (
	ActionUseStatic   FetchAction = iota // The static content is good. Use it.
	ActionRetryOneOff                    // It looks empty/suspicious. Retry with Chrome, but don't ban the domain.
	ActionMarkDynamic                    // It explicitly asked for JS. Retry with Chrome AND ban the domain.
)

type Parser struct {
	UserAgent     string
	allocCtx      context.Context
	domainManager *DomainManager
}

func NewParser(userAgent string, allocCtx context.Context, domainMgr *DomainManager) *Parser {
	return &Parser{UserAgent: userAgent, allocCtx: allocCtx, domainManager: domainMgr}
}

func (p *Parser) GetOutBoundLinks(targetURL string) ([]string, error) {

	body, _, err := p.FetchDynamic(targetURL)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	outBoundLinks, err := p.extractOutBoundLinks(body, targetURL)
	if err != nil {
		return nil, err
	}

	return outBoundLinks, nil
}

func (p *Parser) Parse(targetURL string) (models.PageData, error) {
	var bodyReader io.ReadCloser
	var statusCode int
	var err error
	var loadTime time.Duration

	start := time.Now()

	// 1. CHECK CACHE: Is this domain permanently marked as dynamic?
	if p.domainManager.NeedsDynamic(targetURL) {
		bodyReader, statusCode, err = p.FetchDynamic(targetURL)
	} else {
		// 2. ATTEMPT STATIC FETCH
		bodyReader, statusCode, err = p.FetchStatic(targetURL)

		// 3. ANALYZE STATIC RESULT
		if err == nil {
			bodyBytes, readErr := io.ReadAll(bodyReader)
			bodyReader.Close()

			if readErr != nil {
				return models.PageData{URL: targetURL}, readErr
			}

			// ASK THE JUDGE: What should we do with this body?
			action := p.decideAction(bodyBytes, statusCode)

			switch action {
			case ActionMarkDynamic:
				fmt.Printf("[SmartParse] HARD trigger for %s. Marking Domain as Dynamic.\n", targetURL)
				p.domainManager.MarkDynamic(targetURL)
				// Fallthrough to retry...
				bodyReader, statusCode, err = p.FetchDynamic(targetURL)

			case ActionRetryOneOff:
				fmt.Printf("[SmartParse] SOFT trigger (length/heuristic) for %s. Retrying Dynamic (One-off).\n", targetURL)
				bodyReader, statusCode, err = p.FetchDynamic(targetURL)

			case ActionUseStatic:
				// It was good! Restore the reader for extraction.
				bodyReader = io.NopCloser(bytes.NewReader(bodyBytes))
			}
		}
	}

	loadTime = time.Since(start)

	if err != nil {
		return models.PageData{URL: targetURL}, err
	}
	defer bodyReader.Close()

	// 4. EXTRACT CONTENT
	data, err := p.Extract(bodyReader, targetURL)
	if err != nil {
		return models.PageData{URL: targetURL, StatusCode: statusCode}, err
	}

	data.LoadTime = loadTime
	data.StatusCode = statusCode

	return data, nil
}

func (p *Parser) FetchStatic(targetURL string) (io.ReadCloser, int, error) {
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

const (
	// Removes the "I am a robot" flag
	scriptStealth = `
		Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
		window.navigator.chrome = { runtime: {} };
	`
	// Forces all links into a clean list for the Go parser
	scriptLinkShim = `(function(){
		window.scrollTo(0, document.body.scrollHeight);
		const links = document.querySelectorAll('a[href]');
		const shim = document.createElement('div');
		shim.id = 'crawler-shim';
		shim.style.display = 'none';
		links.forEach(l => {
			const a = document.createElement('a');
			a.href = l.href;
			a.innerText = 'shim-link';
			shim.appendChild(a);
		});
		document.body.appendChild(shim);
	})()`
)

type BrowserProfile struct {
	UserAgent  string
	ClientHint string
}

// Valid Linux Profiles (Matches your Docker Container)
var linuxProfiles = []BrowserProfile{
	// Profile 1: Chrome 132 (Latest)
	{
		UserAgent:  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36",
		ClientHint: `"Not A(Brand";v="99", "Google Chrome";v="132", "Chromium";v="132"`,
	},
	// Profile 2: Chrome 131
	{
		UserAgent:  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		ClientHint: `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`,
	},
	// Profile 3: Chrome 130
	{
		UserAgent:  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
		ClientHint: `"Chromium";v="130", "Google Chrome";v="130", "Not?A_Brand";v="99"`,
	},
}

func getRandomProfile() BrowserProfile {
	return linuxProfiles[rand.Intn(len(linuxProfiles))]
}

func (p *Parser) FetchDynamic(targetURL string) (io.ReadCloser, int, error) {
	profile := getRandomProfile()
	// 1. Determine the correct "Wait Selector" based on the domain
	waitSelector := "body" // Default fallback
	if strings.Contains(targetURL, "newegg.com") {
		waitSelector = "a.item-title"
	} else if strings.Contains(targetURL, "amazon.com") {
		// Amazon's main search result container
		waitSelector = "div.s-main-slot"
	} else if strings.Contains(targetURL, "bestbuy.com") {
		// BestBuy's product item class
		waitSelector = ".sku-item"
	}
	if waitSelector == "APPLES" {
		return nil, 0, nil
	}

	ctx, cancelCtx := chromedp.NewContext(p.allocCtx,
		chromedp.WithLogf(func(string, ...interface{}) {}),
	)
	defer cancelCtx()

	// 4. Timeout (45s)
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var htmlContent string
	var pageTitle string
	var pageText string

	// 5. Run Tasks
	err := chromedp.Run(ctx,
		// (A) Inject Stealth (Pre-load)
		chromedp.ActionFunc(func(c context.Context) error {
			_, err := page.AddScriptToEvaluateOnNewDocument(scriptStealth).Do(c)
			return err
		}),

		network.Enable(),
		network.SetExtraHTTPHeaders(network.Headers{
			"Accept-Language": "en-US,en;q=0.9",

			// USE THE MATCHING HINT HERE:
			"Sec-Ch-Ua": profile.ClientHint,

			"Sec-Ch-Ua-Mobile":          "?0",
			"Sec-Ch-Ua-Platform":        `"Linux"`, // Always Linux for Docker
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
			"Sec-Fetch-User":            "?1",
			"Upgrade-Insecure-Requests": "1",
		}),

		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate(targetURL),

		// 1. Move to a RANDOM point in the "Safe Zone"
		// We target a box between X:300-500 and Y:300-500
		chromedp.ActionFunc(func(c context.Context) error {
			// Randomize X and Y by adding a random number between 0-200
			x := 300 + rand.Intn(200)
			y := 300 + rand.Intn(200)

			// Move the mouse to this random spot
			return chromedp.MouseClickXY(float64(x), float64(y)).Do(c)
		}),

		// 2. Add "Human Jitter" (Small pauses)
		// Humans don't click instantly. They hover, then click.
		chromedp.Sleep(time.Duration(rand.Intn(1000)+500)*time.Millisecond),

		// 3. Scroll randomly (Reading behavior)
		chromedp.ActionFunc(func(c context.Context) error {
			scrollDistance := 300 + rand.Intn(400) // Scroll between 300px and 700px
			script := fmt.Sprintf("window.scrollTo({top: %d, behavior: 'smooth'});", scrollDistance)
			_, err := page.AddScriptToEvaluateOnNewDocument(script).Do(c)
			return err
		}),

		//chromedp.Sleep(5*time.Second),

		chromedp.Evaluate(`document.title`, &pageTitle),
		chromedp.Evaluate(`document.body.innerText.substring(0, 150).replace(/\n/g, " ")`, &pageText),

		// (C) Run the Link Shim
		chromedp.Evaluate(scriptLinkShim, nil),

		// (D) Capture HTML
		chromedp.OuterHTML(`html`, &htmlContent),
	)

	if err != nil {
		return nil, 0, err
	}

	fmt.Printf("\n--- CRAWLER REPORT ---\n")
	fmt.Printf("URL: %s\n", targetURL)
	fmt.Printf("PAGE TITLE:  [%s]\n", pageTitle)
	fmt.Printf("PAGE TEXT:   [%s]\n", pageText)
	fmt.Printf("----------------------\n\n")

	return io.NopCloser(strings.NewReader(htmlContent)), 200, nil
}

func (p *Parser) decideAction(html []byte, statusCode int) FetchAction {
	// 1. Valid HTTP Errors (403/429/500) are NOT fixed by Chrome.
	if statusCode >= 400 {
		return ActionUseStatic
	}

	s := string(html)

	// 2. Hard Failures (Explicit "Enable JS" or Bot Checks)
	// These mean the domain is hostile to static crawlers.
	if strings.Contains(s, "challenge-platform") ||
		strings.Contains(s, "Cloudflare") ||
		strings.Contains(s, "You need to enable JavaScript") ||
		strings.Contains(s, "This site requires Javascript") {
		return ActionMarkDynamic
	}

	// 3. Soft Failures (Suspiciously Empty)
	// This might just be a glitch or a specific page structure.
	// We retry this request, but we don't condemn the whole domain yet.
	if len(s) < 500 {
		return ActionRetryOneOff
	}

	// 4. Success
	return ActionUseStatic
}

func (p *Parser) extractOutBoundLinks(r io.Reader, baseURL string) ([]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	var links []string

	nodeCount := 0 // <--- Debug counter

	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		nodeCount++ // Count every node visited

		if n.Type == html.ElementNode && n.Data == "a" {
			// fmt.Println("Found anchor tag!") // Uncomment to see every <a> tag
			for _, a := range n.Attr {
				if a.Key == "href" {
					// Debug: Print the RAW href value
					// fmt.Printf("  Raw href: %s\n", a.Val)

					absoluteURL := resolveURL(baseURL, a.Val)

					// Debug: Print why it might be failing
					if absoluteURL == "" {
						// fmt.Printf("  -> Failed to resolve: %s with base %s\n", a.Val, baseURL)
					} else {
						links = append(links, absoluteURL)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)

	// CRITICAL CHECK
	if nodeCount < 5 {
		fmt.Println("WARNING: Parser saw almost no nodes! The Reader passed to extractOutBoundLinks was likely empty/already read.")
	} else {
		fmt.Printf("Parser visited %d nodes but found %d links.\n", nodeCount, len(links))
	}

	return links, nil
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
		if n.Type == html.ElementNode && n.Data == "head" {
			return
		}

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
