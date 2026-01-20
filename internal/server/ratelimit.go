package server

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*clientRequests
	limit    int
	window   time.Duration
	cleanup  time.Duration
}

type clientRequests struct {
	count     int
	windowEnd time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientRequests),
		limit:    limit,
		window:   window,
		cleanup:  window * 2,
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, req := range rl.requests {
			if now.After(req.windowEnd) {
				delete(rl.requests, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	req, exists := rl.requests[ip]

	if !exists || now.After(req.windowEnd) {
		rl.requests[ip] = &clientRequests{
			count:     1,
			windowEnd: now.Add(rl.window),
		}
		return true
	}

	if req.count >= rl.limit {
		return false
	}

	req.count++
	return true
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		if !rl.Allow(ip) {
			jsonError(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// Stricter rate limiter for auth endpoints
func NewAuthRateLimiter() *RateLimiter {
	// 5 requests per minute for login/register
	return NewRateLimiter(5, time.Minute)
}

// General rate limiter for API endpoints
func NewAPIRateLimiter() *RateLimiter {
	// 100 requests per minute
	return NewRateLimiter(100, time.Minute)
}
