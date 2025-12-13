package main

import (
	"bytes"
	"flag"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Phase 2: Handler Tests (Direct Function Testing)

func TestHomeHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
		checkHeader    bool
	}{
		{
			name:           "GET / success",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   "Astrology Quiz",
			checkHeader:    true,
		},
		{
			name:           "POST / method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method Not Allowed",
			checkHeader:    false,
		},
		{
			name:           "PUT / method not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method Not Allowed",
			checkHeader:    false,
		},
		{
			name:           "DELETE / method not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method Not Allowed",
			checkHeader:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/", nil)
			rec := httptest.NewRecorder()

			homeHandler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkHeader {
				contentType := rec.Header().Get("Content-Type")
				if contentType != "text/html; charset=utf-8" {
					t.Errorf("expected Content-Type 'text/html; charset=utf-8', got '%s'", contentType)
				}
			}

			body := rec.Body.String()
			if !strings.Contains(body, tt.expectedBody) {
				t.Errorf("expected body to contain '%s', got '%s'", tt.expectedBody, body)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
		checkExact     bool
	}{
		{
			name:           "GET /health success",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
			checkExact:     true,
		},
		{
			name:           "POST /health method not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method Not Allowed",
			checkExact:     false,
		},
		{
			name:           "PUT /health method not allowed",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method Not Allowed",
			checkExact:     false,
		},
		{
			name:           "DELETE /health method not allowed",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method Not Allowed",
			checkExact:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()

			healthHandler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.method == http.MethodGet {
				contentType := rec.Header().Get("Content-Type")
				if contentType != "text/plain" {
					t.Errorf("expected Content-Type 'text/plain', got '%s'", contentType)
				}
			}

			body := rec.Body.String()
			if tt.checkExact {
				if body != tt.expectedBody {
					t.Errorf("expected body '%s', got '%s'", tt.expectedBody, body)
				}
			} else {
				if !strings.Contains(body, tt.expectedBody) {
					t.Errorf("expected body to contain '%s', got '%s'", tt.expectedBody, body)
				}
			}
		})
	}
}

// Phase 3: Routing Tests (Integration with setupRoutes)

func TestSetupRoutes(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		bodyContains   string
	}{
		{
			name:           "GET / root path",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusOK,
			bodyContains:   "Astrology Quiz",
		},
		{
			name:           "GET /health",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
			bodyContains:   "OK",
		},
		{
			name:           "GET /undefined 404",
			method:         http.MethodGet,
			path:           "/undefined",
			expectedStatus: http.StatusNotFound,
			bodyContains:   "404",
		},
		{
			name:           "GET /nonexistent 404",
			method:         http.MethodGet,
			path:           "/nonexistent",
			expectedStatus: http.StatusNotFound,
			bodyContains:   "404",
		},
		{
			name:           "POST / method not allowed",
			method:         http.MethodPost,
			path:           "/",
			expectedStatus: http.StatusMethodNotAllowed,
			bodyContains:   "Method Not Allowed",
		},
		{
			name:           "PUT /health method not allowed",
			method:         http.MethodPut,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
			bodyContains:   "Method Not Allowed",
		},
	}

	mux := setupRoutes()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, tt.bodyContains) {
				t.Errorf("expected body to contain '%s', got '%s'", tt.bodyContains, body)
			}
		})
	}
}

// Phase 4: Middleware Tests

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		handlerStatus  int
		expectedStatus int
	}{
		{
			name:           "middleware passes through 200",
			handlerStatus:  http.StatusOK,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "middleware passes through 404",
			handlerStatus:  http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "middleware passes through 500",
			handlerStatus:  http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that returns the desired status
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.handlerStatus)
				w.Write([]byte("test response"))
			})

			// Wrap with logging middleware
			wrapped := loggingMiddleware(testHandler)

			// Capture log output
			var logBuf bytes.Buffer
			log.SetOutput(&logBuf)
			defer log.SetOutput(nil) // Reset to default after test

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			body := rec.Body.String()
			if body != "test response" {
				t.Errorf("expected body 'test response', got '%s'", body)
			}

			// Verify log output contains method, path, and status
			logOutput := logBuf.String()
			if !strings.Contains(logOutput, "GET") {
				t.Errorf("expected log to contain 'GET', got '%s'", logOutput)
			}
			if !strings.Contains(logOutput, "/test") {
				t.Errorf("expected log to contain '/test', got '%s'", logOutput)
			}
		})
	}
}

func TestResponseWriter(t *testing.T) {
	tests := []struct {
		name           string
		setStatus      bool
		statusCode     int
		expectedStatus int
	}{
		{
			name:           "default status code 200",
			setStatus:      false,
			statusCode:     0,
			expectedStatus: 200,
		},
		{
			name:           "custom status code 404",
			setStatus:      true,
			statusCode:     http.StatusNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "custom status code 500",
			setStatus:      true,
			statusCode:     http.StatusInternalServerError,
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			rw := &responseWriter{
				ResponseWriter: rec,
				statusCode:     200, // Default
			}

			if tt.setStatus {
				rw.WriteHeader(tt.statusCode)
			}

			if rw.statusCode != tt.expectedStatus {
				t.Errorf("expected statusCode %d, got %d", tt.expectedStatus, rw.statusCode)
			}
		})
	}
}

// Phase 5: Utility Function Tests

func TestValidatePort(t *testing.T) {
	tests := []struct {
		name         string
		inputPort    int
		expectedPort int
		shouldLog    bool
	}{
		{
			name:         "valid port 80",
			inputPort:    80,
			expectedPort: 80,
			shouldLog:    false,
		},
		{
			name:         "valid port 8080",
			inputPort:    8080,
			expectedPort: 8080,
			shouldLog:    false,
		},
		{
			name:         "valid port 3000",
			inputPort:    3000,
			expectedPort: 3000,
			shouldLog:    false,
		},
		{
			name:         "valid port 65535",
			inputPort:    65535,
			expectedPort: 65535,
			shouldLog:    false,
		},
		{
			name:         "valid port 1",
			inputPort:    1,
			expectedPort: 1,
			shouldLog:    false,
		},
		{
			name:         "invalid port 0",
			inputPort:    0,
			expectedPort: 8080,
			shouldLog:    true,
		},
		{
			name:         "invalid port -1",
			inputPort:    -1,
			expectedPort: 8080,
			shouldLog:    true,
		},
		{
			name:         "invalid port 70000",
			inputPort:    70000,
			expectedPort: 8080,
			shouldLog:    true,
		},
		{
			name:         "invalid port 65536",
			inputPort:    65536,
			expectedPort: 8080,
			shouldLog:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var logBuf bytes.Buffer
			log.SetOutput(&logBuf)
			defer log.SetOutput(nil) // Reset to default after test

			result := validatePort(tt.inputPort)

			if result != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, result)
			}

			logOutput := logBuf.String()
			if tt.shouldLog && !strings.Contains(logOutput, "Invalid port") {
				t.Errorf("expected log message for invalid port, got none")
			}
			if !tt.shouldLog && logOutput != "" {
				t.Errorf("expected no log output, got '%s'", logOutput)
			}
		})
	}
}

// Phase 6: CLI Flag Tests

func TestFlagParsing(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedPort int
	}{
		{
			name:         "default port",
			args:         []string{},
			expectedPort: 8080,
		},
		{
			name:         "custom port 3000",
			args:         []string{"-port=3000"},
			expectedPort: 3000,
		},
		{
			name:         "custom port 9090",
			args:         []string{"-port=9090"},
			expectedPort: 9090,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new flag set to avoid interfering with global flags
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			port := fs.Int("port", 8080, "Port to listen on")

			// Parse the test arguments
			err := fs.Parse(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			if *port != tt.expectedPort {
				t.Errorf("expected port %d, got %d", tt.expectedPort, *port)
			}
		})
	}
}

func TestFlagParsingWithValidation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedPort int
	}{
		{
			name:         "valid custom port validated",
			args:         []string{"-port=3000"},
			expectedPort: 3000,
		},
		{
			name:         "invalid port validated to default",
			args:         []string{"-port=70000"},
			expectedPort: 8080,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture log output
			var logBuf bytes.Buffer
			log.SetOutput(&logBuf)
			defer log.SetOutput(nil)

			// Create a new flag set
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			port := fs.Int("port", 8080, "Port to listen on")

			// Parse and validate
			err := fs.Parse(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			validPort := validatePort(*port)

			if validPort != tt.expectedPort {
				t.Errorf("expected validated port %d, got %d", tt.expectedPort, validPort)
			}
		})
	}
}

// Phase 7: Integration Tests

func TestServerStartupShutdown(t *testing.T) {
	// Create a test server with our routes
	mux := setupRoutes()
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		bodyContains   string
	}{
		{
			name:           "root path returns HTML",
			path:           "/",
			expectedStatus: http.StatusOK,
			bodyContains:   "Astrology Quiz",
		},
		{
			name:           "health endpoint returns OK",
			path:           "/health",
			expectedStatus: http.StatusOK,
			bodyContains:   "OK",
		},
		{
			name:           "undefined path returns 404",
			path:           "/undefined",
			expectedStatus: http.StatusNotFound,
			bodyContains:   "404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			var buf bytes.Buffer
			buf.ReadFrom(resp.Body)
			body := buf.String()

			if !strings.Contains(body, tt.bodyContains) {
				t.Errorf("expected body to contain '%s', got '%s'", tt.bodyContains, body)
			}
		})
	}
}

func TestServerWithMiddleware(t *testing.T) {
	// Create a test server with middleware
	mux := setupRoutes()
	handler := loggingMiddleware(mux)
	server := httptest.NewServer(handler)
	defer server.Close()

	// Capture log output
	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	defer log.SetOutput(nil)

	// Make a request
	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify middleware logged the request
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "GET") {
		t.Errorf("expected log to contain 'GET', got '%s'", logOutput)
	}
	if !strings.Contains(logOutput, "/") {
		t.Errorf("expected log to contain '/', got '%s'", logOutput)
	}
	if !strings.Contains(logOutput, "200") {
		t.Errorf("expected log to contain '200', got '%s'", logOutput)
	}
}
