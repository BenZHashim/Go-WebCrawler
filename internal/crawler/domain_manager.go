package crawler

import (
	"github.com/temoto/robotstxt"
	"go-crawler/pkg/models"
	"golang.org/x/net/context"
	"golang.org/x/time/rate"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type DomainManager struct {
	mu           sync.RWMutex
	limiters     map[string]*rate.Limiter
	robotsCache  map[string]*robotstxt.Group
	dynamicRules map[string]bool
	fireDelay    time.Duration
}

func NewDomainManager(duration time.Duration) *DomainManager {
	return &DomainManager{
		limiters:     make(map[string]*rate.Limiter),
		robotsCache:  make(map[string]*robotstxt.Group),
		dynamicRules: make(map[string]bool),
		fireDelay:    duration,
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
		limiter = rate.NewLimiter(rate.Every(d.fireDelay), 1)
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
	host := u.Host

	d.mu.RLock()
	group, exists := d.robotsCache[host]
	d.mu.RUnlock()

	if exists {
		if group == nil {
			return true
		}
		return group.Test(u.Path)
	}

	resp, err := http.Get(u.Scheme + "://" + host + "/robots.txt")

	var newGroup *robotstxt.Group
	// Only parse if the request actually succeeded
	if err == nil && resp.StatusCode == 200 {
		data, err := robotstxt.FromResponse(resp)
		if err == nil {
			newGroup = data.FindGroup("MyGoCrawler")
		}
		resp.Body.Close()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if cachedGroup, alreadyFilled := d.robotsCache[host]; alreadyFilled {
		if cachedGroup == nil {
			return true
		}
		return cachedGroup.Test(u.Path)
	}

	d.robotsCache[host] = newGroup

	if newGroup == nil {
		return true
	}
	return newGroup.Test(u.Path)
}

func (d *DomainManager) NeedsDynamic(targetURl string) bool {
	u, _ := url.Parse(targetURl)
	host := u.Host

	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.dynamicRules[host]
}

func (d *DomainManager) MarkDynamic(targetURL string) {
	u, _ := url.Parse(targetURL)
	host := u.Host

	d.mu.Lock()
	// Optimization: If we already know, don't hit DB
	if d.dynamicRules[host] {
		d.mu.Unlock()
		return
	}
	d.dynamicRules[host] = true
	d.mu.Unlock()
}

func getDomain(url string) models.DataSource {
	if strings.Contains(url, "amazon.com") {
		return models.Amazon
	}
	if strings.Contains(url, "newegg.com") {
		return models.Newegg
	}

	return models.None

}
