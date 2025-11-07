package middleware

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// ResponseRecorder wraps http.ResponseWriter to capture status and body
type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
	Body       *bytes.Buffer
}

func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		StatusCode:     200,
		Body:           &bytes.Buffer{},
	}
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *ResponseRecorder) Write(b []byte) (int, error) {
	r.Body.Write(b)
	return r.ResponseWriter.Write(b)
}

// isStaticAsset checks if the request path is for a static asset
func isStaticAsset(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	staticExtensions := []string{
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp",
		".css", ".js", ".map",
		".woff", ".woff2", ".ttf", ".eot", ".otf",
		".pdf", ".zip", ".tar", ".gz",
		".mp4", ".webm", ".ogg", ".mp3", ".wav",
	}
	for _, staticExt := range staticExtensions {
		if ext == staticExt {
			return true
		}
	}
	return false
}

// LoggingMiddleware logs all HTTP requests and responses
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip detailed logging for static assets
		if isStaticAsset(r.URL.Path) {
			recorder := NewResponseRecorder(w)
			next.ServeHTTP(recorder, r)
			duration := time.Since(start)
			log.Printf("%s %s → %d (%s)", r.Method, r.URL.Path, recorder.StatusCode, duration)
			return
		}

		// Log request (metadata only, no body)
		log.Printf("→ %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)

		// Capture response
		recorder := NewResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		// Log response (metadata only, no body)
		duration := time.Since(start)
		log.Printf("← %s %s → %d (%s)", r.Method, r.URL.Path, recorder.StatusCode, duration)
		log.Println()
	})
}

// CompactLoggingMiddleware logs requests in a single line (like nginx)
func CompactLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := NewResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		// Single line log format
		log.Printf("%s %s %d %s %s",
			r.Method,
			r.URL.Path,
			recorder.StatusCode,
			time.Since(start),
			r.RemoteAddr,
		)
	})
}

// StructuredLoggingMiddleware logs in JSON format
func StructuredLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		recorder := NewResponseRecorder(w)
		next.ServeHTTP(recorder, r)

		// Create log entry (metadata only, no bodies)
		logEntry := map[string]interface{}{
			"timestamp":      start.Format(time.RFC3339),
			"method":         r.Method,
			"path":           r.URL.Path,
			"status":         recorder.StatusCode,
			"duration_ms":    time.Since(start).Milliseconds(),
			"remote_addr":    r.RemoteAddr,
			"user_agent":     r.UserAgent(),
			"content_length": r.ContentLength,
		}

		logJSON, _ := json.Marshal(logEntry)
		log.Println(string(logJSON))
	})
}

// formatJSON pretty-prints JSON if valid, otherwise returns original string
func formatJSON(data []byte) string {
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err == nil {
		pretty, err := json.MarshalIndent(obj, "", "  ")
		if err == nil {
			return string(pretty)
		}
	}
	return string(data)
}
