package crawler

import (
	"github.com/temoto/robotstxt"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type DomainManager struct {
	mu          sync.Mutex
	limiters    map[string]*rate.Limiter
	robotsCache map[string]*robotstxt.Group
}

func NewDomainManager() *DomainManager {
	return &DomainManager{
		limiters:    make(map[string]*rate.Limiter),
		robotsCache: make(map[string]*robotstxt.Group),
	}
}

func (d *DomainManager) Wait(targetURL string) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return err
	}
	domain := u.Host

	d.mu.Lock()
	// Check if we already have a limiter for this domain
	limiter, exists := d.limiters[domain]
	if !exists {
		// Create a new limiter: 1 request every 2 seconds
		// rate.Every(2 * time.Second) = interval
		// 1 = burst size (allow 1 request immediately, then wait)
		limiter = rate.NewLimiter(rate.Every(2*time.Second), 1)
		d.limiters[domain] = limiter
	}
	d.mu.Unlock()

	// This blocks the calling goroutine until the limiter allows it to proceed
	return limiter.Wait(context.Background())
}

func (d *DomainManager) IsAllowed(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if we have cached robots.txt data
	group, exists := d.robotsCache[u.Host]
	if !exists {
		// If not, fetch it (simplification: fetching inside lock is slow,
		// in production use a separate fetching routine)
		resp, err := http.Get(u.Scheme + "://" + u.Host + "/robots.txt")
		if err != nil || resp.StatusCode != 200 {
			// Assume allowed if error
			d.robotsCache[u.Host] = nil
			return true
		}
		defer resp.Body.Close()

		data, err := robotstxt.FromResponse(resp)
		if err != nil {
			d.robotsCache[u.Host] = nil
			return true
		}
		group = data.FindGroup("MyGoCrawler")
		d.robotsCache[u.Host] = group
	}

	if group == nil {
		return true // No robots.txt or parse error = Allowed
	}
	return group.Test(u.Path)
}
