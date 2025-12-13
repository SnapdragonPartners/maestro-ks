package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed home.html
var homeHTML string

//go:embed quiz.html
var quizHTML string

//go:embed results.html
var resultsHTML string

//go:embed leaderboard.html
var leaderboardHTML string

// Question represents an astrology trivia question
type Question struct {
	ID          string   `json:"id"`
	Question    string   `json:"question"`
	Choices     []string `json:"choices"`
	AnswerIndex int      `json:"answer_index"`
	Explanation string   `json:"explanation"`
}

// QuizState represents the client-side quiz state
type QuizState struct {
	QuestionIDs  []string `json:"question_ids"`
	CurrentIndex int      `json:"current_index"`
	Score        int      `json:"score"`
}

// LeaderboardEntry represents a single leaderboard entry
type LeaderboardEntry struct {
	Name  string    `json:"name"`
	Score int       `json:"score"`
	Total int       `json:"total"`
	When  time.Time `json:"when"`
}

// LeaderboardManager manages the leaderboard with thread-safe access
type LeaderboardManager struct {
	mu      sync.Mutex
	entries []LeaderboardEntry
}

// Constants for leaderboard configuration
const (
	MaxLeaderboardSize   = 20
	leaderboardFilename  = "leaderboard.json"
	NumQuestions         = 3  // Number of questions per quiz
)

// Global instances
var (
	leaderboardManager LeaderboardManager
	questions []Question
)

// hmacSecret is used to sign and verify quiz state
// In production, this should be loaded from environment variables
const hmacSecret = "astrology-quiz-secret-key-change-in-production"

// signQuizState generates an HMAC-SHA256 signature for a quiz state
func signQuizState(state QuizState) (string, error) {
	// Serialize state to JSON
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("failed to marshal state: %w", err)
	}

	// Create HMAC-SHA256 hash
	mac := hmac.New(sha256.New, []byte(hmacSecret))
	mac.Write(stateJSON)

	// Compute signature and encode to hex
	signature := mac.Sum(nil)
	return hex.EncodeToString(signature), nil
}

// verifyQuizState verifies the HMAC signature and returns the deserialized state
func verifyQuizState(stateJSON string, signature string) (*QuizState, bool) {
	// Decode the provided signature from hex
	providedSig, err := hex.DecodeString(signature)
	if err != nil {
		return nil, false
	}

	// Re-compute HMAC for the received JSON
	mac := hmac.New(sha256.New, []byte(hmacSecret))
	mac.Write([]byte(stateJSON))
	expectedSig := mac.Sum(nil)

	// Compare signatures using constant-time comparison
	if !hmac.Equal(providedSig, expectedSig) {
		return nil, false
	}

	// Deserialize JSON to QuizState
	var state QuizState
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, false
	}

	return &state, true
}

// loadQuestions loads questions from a JSON file and validates them
func loadQuestions(filename string) ([]Question, error) {
	// Read the file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read questions file: %w", err)
	}

	// Parse JSON
	var questions []Question
	if err := json.Unmarshal(data, &questions); err != nil {
		return nil, fmt.Errorf("failed to parse questions JSON: %w", err)
	}

	// Validate each question
	for i, q := range questions {
		if q.AnswerIndex < 0 {
			return nil, fmt.Errorf("question %d (id: %s) has invalid answer_index: %d (must be >= 0)", i, q.ID, q.AnswerIndex)
		}
		if q.AnswerIndex >= len(q.Choices) {
			return nil, fmt.Errorf("question %d (id: %s) has invalid answer_index: %d (must be < %d choices)", i, q.ID, q.AnswerIndex, len(q.Choices))
		}
	}

	return questions, nil
}

// loadLeaderboard loads leaderboard entries from the JSON file
func loadLeaderboard() error {
	// Read the file
	data, err := os.ReadFile(leaderboardFilename)
	if err != nil {
		// If file doesn't exist, create an empty file
		if os.IsNotExist(err) {
			emptyData := []byte("[]")
			if err := os.WriteFile(leaderboardFilename, emptyData, 0644); err != nil {
				return fmt.Errorf("failed to create leaderboard file: %w", err)
			}
			leaderboardManager.entries = []LeaderboardEntry{}
			return nil
		}
		return fmt.Errorf("failed to read leaderboard file: %w", err)
	}

	// Parse JSON
	var entries []LeaderboardEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse leaderboard JSON: %w", err)
	}

	leaderboardManager.entries = entries
	return nil
}

// saveLeaderboard persists the current leaderboard entries to the JSON file
func saveLeaderboard() error {
	// Marshal entries to JSON with indentation
	data, err := json.MarshalIndent(leaderboardManager.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal leaderboard: %w", err)
	}

	// Write to file
	if err := os.WriteFile(leaderboardFilename, data, 0644); err != nil {
		return fmt.Errorf("failed to write leaderboard file: %w", err)
	}

	return nil
}

// saveScore adds a new score to the leaderboard in a thread-safe manner
func saveScore(name string, score int, total int) error {
	leaderboardManager.mu.Lock()
	defer leaderboardManager.mu.Unlock()

	// Create new entry with current timestamp
	entry := LeaderboardEntry{
		Name:  name,
		Score: score,
		Total: total,
		When:  time.Now(),
	}

	// Append to entries
	leaderboardManager.entries = append(leaderboardManager.entries, entry)

	// Sort entries: Score DESC (higher first), When ASC (earlier first for same score)
	sort.Slice(leaderboardManager.entries, func(i, j int) bool {
		if leaderboardManager.entries[i].Score != leaderboardManager.entries[j].Score {
			return leaderboardManager.entries[i].Score > leaderboardManager.entries[j].Score
		}
		return leaderboardManager.entries[i].When.Before(leaderboardManager.entries[j].When)
	})

	// Truncate to MaxLeaderboardSize if necessary
	if len(leaderboardManager.entries) > MaxLeaderboardSize {
		leaderboardManager.entries = leaderboardManager.entries[:MaxLeaderboardSize]
	}

	// Persist to file
	return saveLeaderboard()
}

// getLeaderboard returns a copy of the current leaderboard entries
func getLeaderboard() []LeaderboardEntry {
	leaderboardManager.mu.Lock()
	defer leaderboardManager.mu.Unlock()

	// Return a copy to prevent external modification
	return append([]LeaderboardEntry{}, leaderboardManager.entries...)
}

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

// QuizPageData represents the data passed to the quiz.html template
type QuizPageData struct {
	Question       Question
	CurrentIndex   int
	TotalQuestions int
	Score          int
	QuizState      string
	Signature      string
}

// quizGetHandler handles GET requests to /quiz and starts a new quiz
func quizGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Select random questions
	numToSelect := NumQuestions
	if len(questions) < NumQuestions {
		numToSelect = len(questions)
	}

	// Create a copy of question indices and shuffle them
	indices := make([]int, len(questions))
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	// Select the first numToSelect questions
	selectedQuestionIDs := make([]string, numToSelect)
	for i := 0; i < numToSelect; i++ {
		selectedQuestionIDs[i] = questions[indices[i]].ID
	}

	// Initialize quiz state
	state := QuizState{
		QuestionIDs:  selectedQuestionIDs,
		CurrentIndex: 0,
		Score:        0,
	}

	// Generate signature
	signature, err := signQuizState(state)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error signing quiz state: %v", err)
		return
	}

	// Serialize state to JSON
	stateJSON, err := json.Marshal(state)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error marshaling quiz state: %v", err)
		return
	}

	// Find the first question
	var currentQuestion Question
	for _, q := range questions {
		if q.ID == selectedQuestionIDs[0] {
			currentQuestion = q
			break
		}
	}

	// Prepare template data
	data := QuizPageData{
		Question:       currentQuestion,
		CurrentIndex:   1, // Display as 1-indexed
		TotalQuestions: numToSelect,
		Score:          0,
		QuizState:      string(stateJSON),
		Signature:      signature,
	}

	// Parse and execute template
	tmpl, err := template.New("quiz").Parse(quizHTML)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error parsing quiz template: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing quiz template: %v", err)
	}
}

// quizPostHandler handles POST requests to /quiz and processes answer submissions
func quizPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	stateJSON := r.FormValue("quizState")
	signature := r.FormValue("signature")
	answerStr := r.FormValue("answer")

	// Verify HMAC signature
	state, valid := verifyQuizState(stateJSON, signature)
	if !valid {
		// Redirect to start over
		http.Redirect(w, r, "/quiz", http.StatusSeeOther)
		return
	}

	// Find current question
	if state.CurrentIndex >= len(state.QuestionIDs) {
		http.Error(w, "Invalid quiz state", http.StatusBadRequest)
		return
	}

	currentQuestionID := state.QuestionIDs[state.CurrentIndex]
	var currentQuestion Question
	found := false
	for _, q := range questions {
		if q.ID == currentQuestionID {
			currentQuestion = q
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Question ID %s not found", currentQuestionID)
		return
	}

	// Check answer if provided
	if answerStr != "" {
		answerIndex, err := strconv.Atoi(answerStr)
		if err == nil && answerIndex == currentQuestion.AnswerIndex {
			state.Score++
		}
	}

	// Update state
	state.CurrentIndex++

	// Check if more questions remain
	if state.CurrentIndex < len(state.QuestionIDs) {
		// Generate new signature for updated state
		newSignature, err := signQuizState(*state)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error signing quiz state: %v", err)
			return
		}

		// Serialize updated state
		newStateJSON, err := json.Marshal(state)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error marshaling quiz state: %v", err)
			return
		}

		// Find next question
		nextQuestionID := state.QuestionIDs[state.CurrentIndex]
		var nextQuestion Question
		found = false
		for _, q := range questions {
			if q.ID == nextQuestionID {
				nextQuestion = q
				found = true
				break
			}
		}

		if !found {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Question ID %s not found", nextQuestionID)
			return
		}

		// Prepare template data for next question
		data := QuizPageData{
			Question:       nextQuestion,
			CurrentIndex:   state.CurrentIndex + 1, // Display as 1-indexed
			TotalQuestions: len(state.QuestionIDs),
			Score:          state.Score,
			QuizState:      string(newStateJSON),
			Signature:      newSignature,
		}

		// Parse and execute template
		tmpl, err := template.New("quiz").Parse(quizHTML)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error parsing quiz template: %v", err)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Error executing quiz template: %v", err)
		}
	} else {
		// Quiz complete - redirect to results
		// Serialize final state
		finalStateJSON, err := json.Marshal(state)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error marshaling final state: %v", err)
			return
		}

		// Generate signature for final state
		finalSignature, err := signQuizState(*state)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Printf("Error signing final state: %v", err)
			return
		}

		// Redirect to results with state
		redirectURL := fmt.Sprintf("/quiz/results?state=%s&signature=%s",
			template.URLQueryEscaper(string(finalStateJSON)),
			template.URLQueryEscaper(finalSignature))
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

// ResultsPageData represents the data passed to the results.html template
type ResultsPageData struct {
	Score      int
	Total      int
	Percentage float64
	QuizState  string
	Signature  string
}

// quizResultsGetHandler handles GET requests to /quiz/results
func quizResultsGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract state and signature from query parameters
	stateJSON := r.URL.Query().Get("state")
	signature := r.URL.Query().Get("signature")

	// Verify HMAC signature
	state, valid := verifyQuizState(stateJSON, signature)
	if !valid {
		// Redirect to start over if tampered
		http.Redirect(w, r, "/quiz", http.StatusSeeOther)
		return
	}

	// Calculate percentage
	total := len(state.QuestionIDs)
	var percentage float64
	if total > 0 {
		percentage = float64(state.Score) / float64(total) * 100.0
	}

	// Prepare template data
	data := ResultsPageData{
		Score:      state.Score,
		Total:      total,
		Percentage: percentage,
		QuizState:  stateJSON,
		Signature:  signature,
	}

	// Parse and execute template
	tmpl, err := template.New("results").Parse(resultsHTML)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error parsing results template: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing results template: %v", err)
	}
}

// quizLeaderboardPostHandler handles POST requests to /quiz/leaderboard
func quizLeaderboardPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Extract form values
	name := r.FormValue("name")
	stateJSON := r.FormValue("quizState")
	signature := r.FormValue("signature")

	// Verify HMAC signature
	state, valid := verifyQuizState(stateJSON, signature)
	if !valid {
		// Redirect to start over if tampered
		http.Redirect(w, r, "/quiz", http.StatusSeeOther)
		return
	}

	// Validate name
	name = strings.TrimSpace(name)
	if len(name) == 0 {
		http.Error(w, "Name cannot be empty", http.StatusBadRequest)
		return
	}
	if len(name) > 20 {
		http.Error(w, "Name must be 20 characters or less", http.StatusBadRequest)
		return
	}

	// Save score to leaderboard
	total := len(state.QuestionIDs)
	if err := saveScore(name, state.Score, total); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error saving score: %v", err)
		return
	}

	// Redirect to leaderboard
	http.Redirect(w, r, "/leaderboard", http.StatusSeeOther)
}


// LeaderboardPageData represents the data passed to the leaderboard.html template
type LeaderboardPageData struct {
	Entries []LeaderboardEntry
}

// leaderboardGetHandler handles GET requests to /leaderboard
func leaderboardGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get leaderboard entries
	entries := getLeaderboard()

	// Prepare template data
	data := LeaderboardPageData{
		Entries: entries,
	}

	// Create template with custom functions
	tmpl := template.New("leaderboard").Funcs(template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"div": func(a, b float64) float64 {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"toFloat": func(i int) float64 {
			return float64(i)
		},
	})

	// Parse template
	tmpl, err := tmpl.Parse(leaderboardHTML)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		log.Printf("Error parsing leaderboard template: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing leaderboard template: %v", err)
	}
}

// setupRoutes configures the HTTP routes
func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Register specific routes first
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/quiz/results", quizResultsGetHandler)
	mux.HandleFunc("/quiz/leaderboard", quizLeaderboardPostHandler)
	mux.HandleFunc("/leaderboard", leaderboardGetHandler)
	mux.HandleFunc("/quiz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			quizGetHandler(w, r)
		} else if r.Method == http.MethodPost {
			quizPostHandler(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})
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

	// Load questions from JSON file
	var err error
	questions, err = loadQuestions("questions.json")
	if err != nil {
		log.Printf("Warning: Failed to load questions: %v", err)
	} else {
		log.Printf("Successfully loaded %d questions", len(questions))
	}

	// Load leaderboard from JSON file
	if err := loadLeaderboard(); err != nil {
		log.Printf("Warning: Failed to load leaderboard: %v", err)
	} else {
		log.Printf("Successfully loaded leaderboard with %d entries", len(leaderboardManager.entries))
	}

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
