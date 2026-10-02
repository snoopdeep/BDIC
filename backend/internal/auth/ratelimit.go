package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limiter is a small fixed-window rate limiter held in memory.
//
// It exists to make password guessing slow, and to stop the public enquiry and
// admission forms being used as a spam cannon. It is per-process: with one API
// process, which is what a school runs, that is exactly right. Behind several
// processes this would need Redis, and that is noted in the deployment plan
// rather than solved prematurely.
type Limiter struct {
	mu       sync.Mutex
	hits     map[string]*window
	limit    int
	window   time.Duration
	lastScan time.Time
}

type window struct {
	count   int
	resetAt time.Time
}

// NewLimiter builds a limiter allowing `limit` requests per `per` duration.
func NewLimiter(limit int, per time.Duration) *Limiter {
	return &Limiter{
		hits:     make(map[string]*window),
		limit:    limit,
		window:   per,
		lastScan: time.Now(),
	}
}

// Allow records an attempt and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	// Sweep expired entries occasionally so a long-running process does not
	// accumulate one map entry per IP address that ever touched it.
	if now.Sub(l.lastScan) > l.window {
		for k, v := range l.hits {
			if now.After(v.resetAt) {
				delete(l.hits, k)
			}
		}
		l.lastScan = now
	}

	existing, found := l.hits[key]
	if !found || now.After(existing.resetAt) {
		l.hits[key] = &window{count: 1, resetAt: now.Add(l.window)}
		return true
	}

	if existing.count >= l.limit {
		return false
	}
	existing.count++
	return true
}

// Reset clears a key, called after a successful sign-in so one person typing
// their password wrong twice is not then locked out for the window.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}

// ClientIP returns the caller's address for rate-limiting purposes.
//
// X-Forwarded-For is only trusted when trustProxy is true, which is set for the
// deployed environment that sits behind a load balancer. Trusting it
// unconditionally would let anyone bypass the limiter by sending a header.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			// The left-most entry is the original client; the rest were added
			// by each hop.
			if first, _, found := strings.Cut(forwarded, ","); found {
				if ip := strings.TrimSpace(first); ip != "" {
					return ip
				}
			} else if ip := strings.TrimSpace(forwarded); ip != "" {
				return ip
			}
		}
		if real := strings.TrimSpace(r.Header.Get("X-Real-IP")); real != "" {
			return real
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
