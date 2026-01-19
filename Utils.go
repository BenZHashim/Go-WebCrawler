package main

import "sync"

type SafeMap struct {
	mu sync.Mutex
	v  map[string]bool
}

func (s *SafeMap) Contains(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.v[url] {
		return true // Already visited
	}
	s.v[url] = true
	return false // New URL
}
