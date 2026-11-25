package api

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// CORS allows the static frontend (e.g. GitHub Pages) to call the API.
// allowed is a list of origins; "*" allows any.
func CORS(allowed []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (slices.Contains(allowed, "*") || slices.Contains(allowed, origin)) {
			// Echo the specific origin instead of "*" and add Vary: Origin, so
			// caches don't serve one site's CORS answer to another site.
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(s int) { w.status = s; w.ResponseWriter.WriteHeader(s) }

// Logging logs each request and turns panics into 500s.
func Logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{w, http.StatusOK}
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic", "path", r.URL.Path, "panic", p)
				writeErr(sw, http.StatusInternalServerError, "internal error")
			}
			if strings.HasPrefix(r.URL.Path, "/api/") {
				log.Info("http", "method", r.Method, "path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds())
			}
		}()
		next.ServeHTTP(sw, r)
	})
}

// RateLimit is a tiny per-IP token bucket for the auth endpoints, to slow
// down password guessing. Good enough for a single instance.
func RateLimit(perMinute int, next http.Handler) http.Handler {
	type bucket struct {
		tokens float64
		last   time.Time
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		// Behind Render/Azure the real client IP is in X-Forwarded-For
		// ("client, proxy1, proxy2"), and we take the first entry. A client
		// can spoof this header when the app is reached directly, so this is a
		// speed bump against password guessing, not a security boundary.
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		ip, _, _ = strings.Cut(ip, ",")
		mu.Lock()
		b, ok := buckets[ip]
		now := time.Now()
		if !ok {
			b = &bucket{float64(perMinute), now}
			buckets[ip] = b
		}
		// Token bucket: refill continuously at perMinute tokens per minute,
		// never above perMinute (the burst size); each request costs one token.
		// Note: the map is never pruned, which is fine for a demo but would
		// need a periodic cleanup under real traffic.
		b.tokens = min(float64(perMinute), b.tokens+now.Sub(b.last).Minutes()*float64(perMinute))
		b.last = now
		allowed := b.tokens >= 1
		if allowed {
			b.tokens--
		}
		mu.Unlock()
		if !allowed {
			writeErr(w, http.StatusTooManyRequests, "too many attempts, try again in a minute")
			return
		}
		next.ServeHTTP(w, r)
	})
}
