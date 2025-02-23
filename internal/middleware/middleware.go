package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Middleware provides HTTP middleware functions
type Middleware struct {
	logger *slog.Logger
}

// NewMiddleware creates a new Middleware instance
func NewMiddleware(logger *slog.Logger) *Middleware {
	return &Middleware{logger: logger}
}

// Apply applies all middleware to the handler
func (m *Middleware) Apply(next http.Handler) http.Handler {
	// Chain middleware in order: CORS -> Logging -> Panic Recovery
	return m.recoverPanic(m.logRequest(m.cors(next)))
}

func (m *Middleware) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("DISABLE_CORS") == "true" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Error("Panic recovered", "error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		m.logger.Info("Request completed", "method", r.Method, "path", r.URL.Path, "duration", duration)
	})
}
