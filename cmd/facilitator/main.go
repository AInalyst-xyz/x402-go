package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/x402-rs/x402-go/pkg/config"
	"github.com/x402-rs/x402-go/pkg/handlers"
	"github.com/x402-rs/x402-go/pkg/middleware"
)

func main() {
	// Configure logging
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize facilitator
	fac, err := cfg.InitializeFacilitator()
	if err != nil {
		log.Fatalf("Failed to initialize facilitator: %v", err)
	}

	// Create HTTP handler
	handler := handlers.NewHandler(fac)

	// Setup routes
	mux := http.NewServeMux()
	handler.SetupRoutes(mux)

	// Serve frontend SPA at "/" from web/dist if it exists
	webDistDir := filepath.Join("web", "dist")
	if stat, err := os.Stat(webDistDir); err == nil && stat.IsDir() {
		fileServer := http.FileServer(http.Dir(webDistDir))
		mux.Handle("/", spaHandler(webDistDir, fileServer))
		log.Printf("Serving frontend SPA from %s at /", webDistDir)
	} else {
		log.Printf("Frontend build directory not found at %s; '/' will not serve the SPA", webDistDir)
	}

	// Add logging middleware based on LOG_FORMAT environment variable
	// Options: "detailed" (default), "compact", "json", "none"
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "detailed"
	}

	var loggedHandler http.Handler
	switch logFormat {
	case "compact":
		log.Println("Using compact logging format")
		loggedHandler = middleware.CompactLoggingMiddleware(mux)
	case "json":
		log.Println("Using JSON structured logging format")
		loggedHandler = middleware.StructuredLoggingMiddleware(mux)
	case "none":
		log.Println("Logging disabled")
		loggedHandler = mux
	default:
		log.Println("Using detailed logging format")
		loggedHandler = middleware.LoggingMiddleware(mux)
	}

	// Add request size limit middleware (1MB limit)
	sizeLimitedHandler := requestSizeLimitMiddleware(loggedHandler, 1<<20) // 1MB

	// Add CORS middleware
	corsHandler := corsMiddleware(sizeLimitedHandler)

	// Create server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      corsHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting x402 facilitator on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// requestSizeLimitMiddleware limits the maximum size of request bodies to prevent DoS attacks
func requestSizeLimitMiddleware(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Limit the request body size
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers to responses
// Uses reflective CORS pattern for public API - allows any origin but without credentials
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" {
			// Reflect the origin back (allows any origin for browser requests)
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		} else {
			// For non-CORS requests (same-origin, curl, postman, etc.)
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Payment-Payload")

		// IMPORTANT: Do NOT set Access-Control-Allow-Credentials
		// This is a public API and should never use credentials

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// spaHandler serves static files if they exist, otherwise falls back to index.html for SPA routing
func spaHandler(root string, fileServer http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // try to serve the requested file from the built assets
        requestedPath := filepath.Clean(r.URL.Path)
        // ensure leading slash is trimmed when joining paths
        candidate := filepath.Join(root, requestedPath)
        if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
            fileServer.ServeHTTP(w, r)
            return
        }
        // fallback to index.html for client-side routes
        http.ServeFile(w, r, filepath.Join(root, "index.html"))
    })
}
