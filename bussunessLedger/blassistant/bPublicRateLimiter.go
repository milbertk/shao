package blassistant

import (
	"sync"
	"time"
)

// dailyCounter counts requests per key for the current day, in memory.
// It protects the public assistant from one IP sending many questions
// (visitors can change their visitor id, but not easily their IP).
// Counts reset every day and when the server restarts.
type dailyCounter struct {
	mu     sync.Mutex
	day    string
	counts map[string]int
}

// maxCounterKeys bounds the memory used by the counter.
const maxCounterKeys = 100000

// allow increments the key's counter and reports whether it is within limit.
func (c *dailyCounter) allow(key string, limit int, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	today := now.Format("2006-01-02")
	if c.day != today || c.counts == nil || len(c.counts) > maxCounterKeys {
		c.day = today
		c.counts = map[string]int{}
	}

	if c.counts[key] >= limit {
		return false
	}
	c.counts[key]++
	return true
}

var publicIPCounter = &dailyCounter{}

// PublicDailyLimiter is the same in-memory daily counter, exported so other
// public endpoints (the store order and booking forms) reuse it
// (2026-10-25, security S4). It protects against one IP sending many
// requests; durable limits (per company / per email) live in the database.
type PublicDailyLimiter struct {
	counter dailyCounter
}

// Allow increments the key's counter and reports whether it is within limit.
func (l *PublicDailyLimiter) Allow(key string, limit int) bool {
	return l.counter.allow(key, limit, time.Now())
}
