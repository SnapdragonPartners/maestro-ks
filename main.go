package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

//go:embed home.html
var homeHTML string

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before writing
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// validatePort ensures port is in valid range (1-65535)
func validatePort(port int) int {
	if port < 1 || port > 65535 {
		log.Printf("Invalid port %d, using default port 8080", port)
		return 8080
	}
	return port
}

// loggingMiddleware logs all HTTP requests with method, path, and status code
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     200, // Default status code
		}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Log the request
		log.Printf("%s %s -> %d", r.Method, r.URL.Path, wrapped.statusCode)
	})
}

// homeHandler serves the embedded HTML home page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(homeHTML))
}

// healthHandler serves the health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// setupRoutes configures the HTTP routes
func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Register specific routes first
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Handle exact root path match
		if r.URL.Path == "/" {
			homeHandler(w, r)
			return
		}
		// All other paths return 404
		http.NotFound(w, r)
	})

	return mux
}

func main() {
	// Parse CLI flags
	port := flag.Int("port", 8080, "Port to listen on")
	flag.Parse()

	// Validate port
	validPort := validatePort(*port)

	// Setup routes
	mux := setupRoutes()

	// Wrap with logging middleware
	handler := loggingMiddleware(mux)

	// Configure HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", validPort),
		Handler: handler,
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on port %d", validPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down gracefully...", sig)

	// Create shutdown context with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	} else {
		log.Println("Server shutdown complete")
	}
}
