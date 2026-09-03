package middleware

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimit caps requests per client IP using a fixed one-minute window
// counted in Redis (INCR + EXPIRE), shared across all replicas of the app.
func RateLimit(cache *redis.Client, requestsPerMinute int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ip := clientIP(r)
			window := time.Now().Unix() / 60
			key := fmt.Sprintf("ratelimit:%s:%d", ip, window)

			count, err := cache.Incr(ctx, key).Result()
			if err != nil {
				// Redis being unavailable should not take the API down.
				next.ServeHTTP(w, r)
				return
			}
			if count == 1 {
				cache.Expire(ctx, key, time.Minute)
			}

			if int(count) > requestsPerMinute {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
