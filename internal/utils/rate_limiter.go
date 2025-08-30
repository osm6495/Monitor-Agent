package utils

import (
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	rate       int
	interval   time.Duration
	tokens     chan struct{}
	mu         sync.Mutex
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		rate:       rate,
		interval:   interval,
		tokens:     make(chan struct{}, rate),
		lastRefill: time.Now(),
	}

	// Fill the token bucket initially
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	// Start the refill goroutine
	go rl.refill()

	return rl
}

// Wait blocks until a token is available
func (rl *RateLimiter) Wait() {
	<-rl.tokens
}

// TryWait attempts to get a token without blocking
func (rl *RateLimiter) TryWait() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// refill refills the token bucket at the specified rate
func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.interval / time.Duration(rl.rate))
	defer ticker.Stop()

	for range ticker.C {
		select {
		case rl.tokens <- struct{}{}:
			// Token added successfully
		default:
			// Bucket is full, skip
		}
	}
}

// GetRate returns the current rate limit
func (rl *RateLimiter) GetRate() int {
	return rl.rate
}

// GetInterval returns the current interval
func (rl *RateLimiter) GetInterval() time.Duration {
	return rl.interval
}

// UpdateRate updates the rate limit
func (rl *RateLimiter) UpdateRate(newRate int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.rate = newRate
}

// UpdateInterval updates the interval
func (rl *RateLimiter) UpdateInterval(newInterval time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.interval = newInterval
}
