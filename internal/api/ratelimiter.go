package api

import (
	"net/http"

	"golang.org/x/time/rate"
)

type rateLimiter interface {
	Allow() bool
}

type limiterAdapter struct {
	limiter *rate.Limiter
}

func newTokenBucketLimiter(ratePerSecond float64, burst int) rateLimiter {
	if ratePerSecond <= 0 {
		ratePerSecond = 1
	}
	if burst <= 0 {
		burst = 1
	}

	return &limiterAdapter{
		limiter: rate.NewLimiter(rate.Limit(ratePerSecond), burst),
	}
}

func (l *limiterAdapter) Allow() bool {
	if l == nil || l.limiter == nil {
		return true
	}
	return l.limiter.Allow()
}

func rateLimitMiddleware(limiter rateLimiter, next http.Handler) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter.Allow() {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusTooManyRequests, "Too many requests", "rate limit exceeded, please retry shortly")
	})
}
