package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"abt-dashboard/internal/config"
	"abt-dashboard/internal/errors"
	"abt-dashboard/internal/observability"
)

type Middleware func(http.Handler) http.Handler

func Chain(middlewares ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}

func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set("X-Request-ID", requestID)
			ctx := observability.WithRequestID(r.Context(), requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Logger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			requestID := observability.GetRequestID(r.Context())

			logger.Info("request started",
				"method", r.Method,
				"url", r.URL.String(),
				"user_agent", r.UserAgent(),
				"remote_addr", r.RemoteAddr,
				"request_id", requestID,
			)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			logger.Info("request completed",
				"method", r.Method,
				"url", r.URL.String(),
				"status", wrapped.statusCode,
				"duration", duration,
				"request_id", requestID,
			)
		})
	}
}

func Tracing() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := observability.StartSpan(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
			defer span.Finish()

			span.SetTag("http.method", r.Method)
			span.SetTag("http.url", r.URL.String())
			span.SetTag("http.user_agent", r.UserAgent())

			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r.WithContext(ctx))

			span.SetTag("http.status_code", strconv.Itoa(wrapped.statusCode))

			if wrapped.statusCode >= 400 {
				span.SetError(fmt.Errorf("HTTP %d", wrapped.statusCode))
			}
		})
	}
}

func CORS(config config.SecurityConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if isAllowedOrigin(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeaders() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline'; connect-src 'self'")

			next.ServeHTTP(w, r)
		})
	}
}

type RateLimiter struct {
	limiters map[string]*rate.Limiter
	config   config.SecurityConfig
	mu       sync.RWMutex
}

func NewRateLimiter(config config.SecurityConfig) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
	}
}

func (rl *RateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter, exists = rl.limiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(rl.config.RateLimitRPS), rl.config.RateLimitBurst)
			rl.limiters[ip] = limiter

			go func() {
				time.Sleep(time.Minute)
				rl.mu.Lock()
				delete(rl.limiters, ip)
				rl.mu.Unlock()
			}()
		}
		rl.mu.Unlock()
	}

	return limiter
}

func (rl *RateLimiter) Allow(ip string) bool {
	if !rl.config.EnableRateLimit {
		return true
	}

	limiter := rl.getLimiter(ip)
	return limiter.Allow()
}

func RateLimit(limiter *RateLimiter, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := getClientIP(r)

			if !limiter.Allow(ip) {
				requestID := observability.GetRequestID(r.Context())
				err := errors.RateLimit("Too many requests")

				logger.Warn("rate limit exceeded",
					"ip", ip,
					"request_id", requestID,
				)

				errors.WriteError(w, logger, err, requestID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func TrustedProxy(config config.SecurityConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isTrustedProxy(r.RemoteAddr, config.TrustedProxies) {
				r.Header.Del("X-Forwarded-For")
				r.Header.Del("X-Real-IP")
				r.Header.Del("X-Forwarded-Proto")
			}

			next.ServeHTTP(w, r)
		})
	}
}

func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := observability.GetRequestID(r.Context())

					logger.Error("panic recovered",
						"error", err,
						"request_id", requestID,
						"method", r.Method,
						"url", r.URL.String(),
					)

					appErr := errors.Internal("An unexpected error occurred")
					errors.WriteError(w, logger, appErr, requestID)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Flush implements http.Flusher if the underlying ResponseWriter does
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func isAllowedOrigin(origin string, allowed []string) bool {
	for _, allowedOrigin := range allowed {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}

func isTrustedProxy(remoteAddr string, trusted []string) bool {
	host, _, _ := net.SplitHostPort(remoteAddr)

	for _, trustedIP := range trusted {
		if trustedIP == host {
			return true
		}
	}
	return false
}
