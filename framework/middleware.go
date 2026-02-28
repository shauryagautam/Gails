package framework

import (
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/shaurya/gails/cache"
	"github.com/shaurya/gails/framework/i18n"
	"go.uber.org/zap"
)

// Logger is structured request logging middleware.
func Logger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				if Log != nil {
					Log.Info("request",
						zap.String("method", r.Method),
						zap.String("path", r.URL.Path),
						zap.Int("status", ww.Status()),
						zap.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000.0),
						zap.String("request_id", middleware.GetReqID(r.Context())),
						zap.String("ip", r.RemoteAddr),
						zap.String("user_agent", r.UserAgent()),
					)
				}
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// Recovery catches panics and renders an error page. Dev mode shows a rich error page.
func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					env := os.Getenv("APP_ENV")
					if env == "" {
						env = "development"
					}
					if env == "development" {
						DevErrorHandler(w, r, err)
					} else {
						ProdErrorHandler(w, r, err)
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID generates a UUID request ID and adds it to context and response header.
func RequestID() func(http.Handler) http.Handler {
	return middleware.RequestID
}

// SecureHeaders adds security-related HTTP headers.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// CORSConfig configures CORS behavior.
type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
	MaxAge       int
}

// CORS is configurable CORS middleware.
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 86400
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowed := false
			for _, o := range config.AllowOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
			w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CSRF implements double-submit cookie CSRF protection, skipped for JSON requests.
func CSRF() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CSRF for JSON API requests
			ct := r.Header.Get("Content-Type")
			if strings.Contains(ct, "application/json") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip safe methods
			if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
				// Set CSRF cookie if not present
				if _, err := r.Cookie("csrf_token"); err != nil {
					token := generateCSRFToken()
					http.SetCookie(w, &http.Cookie{
						Name:     "csrf_token",
						Value:    token,
						Path:     "/",
						HttpOnly: false, // Readable by JS for AJAX
						SameSite: http.SameSiteLaxMode,
					})
				}
				next.ServeHTTP(w, r)
				return
			}

			// Validate CSRF for unsafe methods
			cookie, err := r.Cookie("csrf_token")
			if err != nil {
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			formToken := r.FormValue("csrf_token")
			if formToken == "" {
				formToken = r.Header.Get("X-CSRF-Token")
			}

			if formToken != cookie.Value {
				http.Error(w, "CSRF token invalid", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// gzipResponseWriter wraps http.ResponseWriter with gzip compression.
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// Compress applies gzip compression for responses over a threshold.
func Compress() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			gz, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			defer gz.Close()

			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")
			next.ServeHTTP(gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
		})
	}
}

// RateLimit implements Redis-backed sliding window rate limiting.
func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	// Fallback in-memory rate limiter when Redis is not available
	var mu sync.Mutex
	counts := make(map[string][]time.Time)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cache.Redis != nil {
				redisRateLimit(w, r, next, limit, window)
				return
			}

			// In-memory fallback
			mu.Lock()
			ip := r.RemoteAddr
			now := time.Now()
			cutoff := now.Add(-window)

			// Clean old entries
			var valid []time.Time
			for _, t := range counts[ip] {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			valid = append(valid, now)
			counts[ip] = valid
			count := len(valid)
			mu.Unlock()

			if count > limit {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Rate limit exceeded"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func redisRateLimit(w http.ResponseWriter, r *http.Request, next http.Handler, limit int, window time.Duration) {
	ctx := r.Context()
	key := fmt.Sprintf("ratelimit:%s", r.RemoteAddr)

	now := time.Now().UnixNano()
	clearBefore := now - window.Nanoseconds()

	pipe := cache.Redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", clearBefore))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: fmt.Sprintf("%d", now)})
	pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window)

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		if Log != nil {
			Log.Error("Rate limit error", zap.Error(err))
		}
		next.ServeHTTP(w, r)
		return
	}

	count := cmds[2].(*redis.IntCmd).Val()
	if count > int64(limit) {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("Rate limit exceeded"))
		return
	}

	next.ServeHTTP(w, r)
}

// Locale detects the user locale from query param, session, or Accept-Language.
func Locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check query param ?locale=
		lang := r.URL.Query().Get("locale")

		// 2. Check Accept-Language header
		if lang == "" {
			accept := r.Header.Get("Accept-Language")
			if accept != "" {
				parts := strings.Split(accept, ",")
				if len(parts) > 0 {
					lang = strings.TrimSpace(strings.Split(parts[0], ";")[0])
					lang = strings.Split(lang, "-")[0]
				}
			}
		}

		if lang != "" {
			i18n.SetLocale(lang)
		}

		next.ServeHTTP(w, r)
	})
}
