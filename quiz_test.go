package main

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestLoadQuestions_Success tests loading valid questions from a file
func TestLoadQuestions_Success(t *testing.T) {
	// Create a temporary test file with valid questions
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_questions.json")

	validJSON := `[
		{
			"id": "q1",
			"question": "Test question 1?",
			"choices": ["A", "B", "C", "D"],
			"answer_index": 0,
			"explanation": "Test explanation 1"
		},
		{
			"id": "q2",
			"question": "Test question 2?",
			"choices": ["W", "X", "Y", "Z"],
			"answer_index": 2,
			"explanation": "Test explanation 2"
		}
	]`

	if err := os.WriteFile(testFile, []byte(validJSON), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Load questions
	questions, err := loadQuestions(testFile)
	if err != nil {
		t.Fatalf("loadQuestions() failed: %v", err)
	}

	// Verify number of questions
	if len(questions) != 2 {
		t.Errorf("expected 2 questions, got %d", len(questions))
	}

	// Verify first question
	if questions[0].ID != "q1" {
		t.Errorf("expected first question ID 'q1', got '%s'", questions[0].ID)
	}
	if questions[0].Question != "Test question 1?" {
		t.Errorf("expected first question text 'Test question 1?', got '%s'", questions[0].Question)
	}
	if len(questions[0].Choices) != 4 {
		t.Errorf("expected 4 choices, got %d", len(questions[0].Choices))
	}
	if questions[0].AnswerIndex != 0 {
		t.Errorf("expected answer_index 0, got %d", questions[0].AnswerIndex)
	}

	// Verify second question
	if questions[1].AnswerIndex != 2 {
		t.Errorf("expected second question answer_index 2, got %d", questions[1].AnswerIndex)
	}
}

// TestLoadQuestions_ValidateAnswerIndex tests answer_index validation
func TestLoadQuestions_ValidateAnswerIndex(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid answer_index within bounds",
			jsonContent: `[{
				"id": "q1",
				"question": "Test?",
				"choices": ["A", "B", "C"],
				"answer_index": 1,
				"explanation": "Test"
			}]`,
			expectError: false,
		},
		{
			name: "valid answer_index at lower bound",
			jsonContent: `[{
				"id": "q1",
				"question": "Test?",
				"choices": ["A", "B", "C"],
				"answer_index": 0,
				"explanation": "Test"
			}]`,
			expectError: false,
		},
		{
			name: "valid answer_index at upper bound",
			jsonContent: `[{
				"id": "q1",
				"question": "Test?",
				"choices": ["A", "B", "C"],
				"answer_index": 2,
				"explanation": "Test"
			}]`,
			expectError: false,
		},
		{
			name: "invalid negative answer_index",
			jsonContent: `[{
				"id": "q1",
				"question": "Test?",
				"choices": ["A", "B", "C"],
				"answer_index": -1,
				"explanation": "Test"
			}]`,
			expectError: true,
			errorMsg:    "must be >= 0",
		},
		{
			name: "invalid answer_index exceeds choices length",
			jsonContent: `[{
				"id": "q1",
				"question": "Test?",
				"choices": ["A", "B", "C"],
				"answer_index": 3,
				"explanation": "Test"
			}]`,
			expectError: true,
			errorMsg:    "must be < 3 choices",
		},
		{
			name: "invalid answer_index far exceeds choices length",
			jsonContent: `[{
				"id": "q1",
				"question": "Test?",
				"choices": ["A", "B"],
				"answer_index": 10,
				"explanation": "Test"
			}]`,
			expectError: true,
			errorMsg:    "must be < 2 choices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.json")

			if err := os.WriteFile(testFile, []byte(tt.jsonContent), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Load questions
			questions, err := loadQuestions(testFile)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(questions) != 1 {
					t.Errorf("expected 1 question, got %d", len(questions))
				}
			}
		})
	}
}

// TestLoadQuestions_FileNotFound tests behavior when file doesn't exist
func TestLoadQuestions_FileNotFound(t *testing.T) {
	// Try to load a non-existent file
	_, err := loadQuestions("/nonexistent/path/questions.json")

	if err == nil {
		t.Error("expected error for non-existent file, got none")
	}

	if !contains(err.Error(), "failed to read questions file") {
		t.Errorf("expected error message about reading file, got: %v", err)
	}
}

// TestLoadQuestions_InvalidJSON tests behavior with malformed JSON
func TestLoadQuestions_InvalidJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
	}{
		{
			name:        "completely invalid JSON",
			jsonContent: `{this is not valid json}`,
		},
		{
			name:        "incomplete JSON",
			jsonContent: `[{"id": "q1", "question": "Test?"`,
		},
		{
			name:        "wrong type",
			jsonContent: `{"not": "an array"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "invalid.json")

			if err := os.WriteFile(testFile, []byte(tt.jsonContent), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Try to load questions
			_, err := loadQuestions(testFile)

			if err == nil {
				t.Error("expected error for invalid JSON, got none")
			}

			if !contains(err.Error(), "failed to parse questions JSON") {
				t.Errorf("expected error message about parsing JSON, got: %v", err)
			}
		})
	}
}

// TestLoadQuestions_EmptyFile tests behavior with empty file/array
func TestLoadQuestions_EmptyFile(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		expectError bool
	}{
		{
			name:        "empty array",
			jsonContent: `[]`,
			expectError: false,
		},
		{
			name:        "empty file",
			jsonContent: ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "empty.json")

			if err := os.WriteFile(testFile, []byte(tt.jsonContent), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			// Try to load questions
			questions, err := loadQuestions(testFile)

			if tt.expectError {
				if err == nil {
					t.Error("expected error for empty file, got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(questions) != 0 {
					t.Errorf("expected 0 questions, got %d", len(questions))
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestSignQuizState_Success tests that signQuizState generates valid signatures
func TestSignQuizState_Success(t *testing.T) {
	state := QuizState{
		QuestionIDs:  []string{"q1", "q2", "q3"},
		CurrentIndex: 1,
		Score:        10,
	}

	signature, err := signQuizState(state)
	if err != nil {
		t.Fatalf("signQuizState() failed: %v", err)
	}

	// Verify signature is non-empty
	if signature == "" {
		t.Error("expected non-empty signature")
	}

	// Verify signature is valid hex (should be 64 hex chars for SHA256)
	if len(signature) != 64 {
		t.Errorf("expected signature length 64, got %d", len(signature))
	}

	// Verify signature is valid hex encoding
	_, err = hex.DecodeString(signature)
	if err != nil {
		t.Errorf("signature is not valid hex: %v", err)
	}

	// Verify signature is deterministic
	signature2, err := signQuizState(state)
	if err != nil {
		t.Fatalf("signQuizState() second call failed: %v", err)
	}
	if signature != signature2 {
		t.Error("expected same signature for same input (deterministic)")
	}
}

// TestSignQuizState_DifferentStates tests that different states produce different signatures
func TestSignQuizState_DifferentStates(t *testing.T) {
	state1 := QuizState{
		QuestionIDs:  []string{"q1", "q2", "q3"},
		CurrentIndex: 0,
		Score:        0,
	}

	state2 := QuizState{
		QuestionIDs:  []string{"q1", "q2", "q3"},
		CurrentIndex: 1,
		Score:        0,
	}

	state3 := QuizState{
		QuestionIDs:  []string{"q1", "q2", "q3"},
		CurrentIndex: 0,
		Score:        5,
	}

	state4 := QuizState{
		QuestionIDs:  []string{"q1", "q2"},
		CurrentIndex: 0,
		Score:        0,
	}

	sig1, err := signQuizState(state1)
	if err != nil {
		t.Fatalf("signQuizState(state1) failed: %v", err)
	}

	sig2, err := signQuizState(state2)
	if err != nil {
		t.Fatalf("signQuizState(state2) failed: %v", err)
	}

	sig3, err := signQuizState(state3)
	if err != nil {
		t.Fatalf("signQuizState(state3) failed: %v", err)
	}

	sig4, err := signQuizState(state4)
	if err != nil {
		t.Fatalf("signQuizState(state4) failed: %v", err)
	}

	// All signatures should be different
	if sig1 == sig2 {
		t.Error("expected different signatures for different CurrentIndex")
	}
	if sig1 == sig3 {
		t.Error("expected different signatures for different Score")
	}
	if sig1 == sig4 {
		t.Error("expected different signatures for different QuestionIDs")
	}
}

// TestVerifyQuizState_ValidSignature tests verification with valid signature
func TestVerifyQuizState_ValidSignature(t *testing.T) {
	originalState := QuizState{
		QuestionIDs:  []string{"q1", "q2", "q3", "q4"},
		CurrentIndex: 2,
		Score:        15,
	}

	// Generate valid signature
	signature, err := signQuizState(originalState)
	if err != nil {
		t.Fatalf("signQuizState() failed: %v", err)
	}

	// Serialize state to JSON
	stateJSON, err := json.Marshal(originalState)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	// Verify the signature
	verifiedState, valid := verifyQuizState(string(stateJSON), signature)

	if !valid {
		t.Error("expected signature to be valid")
	}

	if verifiedState == nil {
		t.Fatal("expected non-nil state")
	}

	// Verify deserialized state matches original
	if verifiedState.CurrentIndex != originalState.CurrentIndex {
		t.Errorf("expected CurrentIndex %d, got %d", originalState.CurrentIndex, verifiedState.CurrentIndex)
	}
	if verifiedState.Score != originalState.Score {
		t.Errorf("expected Score %d, got %d", originalState.Score, verifiedState.Score)
	}
	if len(verifiedState.QuestionIDs) != len(originalState.QuestionIDs) {
		t.Errorf("expected %d QuestionIDs, got %d", len(originalState.QuestionIDs), len(verifiedState.QuestionIDs))
	}
	for i, qid := range originalState.QuestionIDs {
		if verifiedState.QuestionIDs[i] != qid {
			t.Errorf("expected QuestionID[%d] = %s, got %s", i, qid, verifiedState.QuestionIDs[i])
		}
	}
}

// TestVerifyQuizState_InvalidSignature tests verification with invalid signatures
func TestVerifyQuizState_InvalidSignature(t *testing.T) {
	state := QuizState{
		QuestionIDs:  []string{"q1", "q2"},
		CurrentIndex: 0,
		Score:        0,
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	tests := []struct {
		name      string
		signature string
	}{
		{
			name:      "completely wrong signature",
			signature: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		{
			name:      "empty signature",
			signature: "",
		},
		{
			name:      "modified signature",
			signature: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifiedState, valid := verifyQuizState(string(stateJSON), tt.signature)

			if valid {
				t.Error("expected signature to be invalid")
			}
			if verifiedState != nil {
				t.Error("expected nil state for invalid signature")
			}
		})
	}
}

// TestVerifyQuizState_TamperedData tests verification with tampered JSON data
func TestVerifyQuizState_TamperedData(t *testing.T) {
	originalState := QuizState{
		QuestionIDs:  []string{"q1", "q2", "q3"},
		CurrentIndex: 1,
		Score:        5,
	}

	// Generate valid signature for original state
	signature, err := signQuizState(originalState)
	if err != nil {
		t.Fatalf("signQuizState() failed: %v", err)
	}

	tests := []struct {
		name         string
		tamperedData string
	}{
		{
			name:         "modified score",
			tamperedData: `{"question_ids":["q1","q2","q3"],"current_index":1,"score":100}`,
		},
		{
			name:         "modified current index",
			tamperedData: `{"question_ids":["q1","q2","q3"],"current_index":2,"score":5}`,
		},
		{
			name:         "added question ID",
			tamperedData: `{"question_ids":["q1","q2","q3","q4"],"current_index":1,"score":5}`,
		},
		{
			name:         "removed question ID",
			tamperedData: `{"question_ids":["q1","q2"],"current_index":1,"score":5}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Try to verify tampered data with original signature
			verifiedState, valid := verifyQuizState(tt.tamperedData, signature)

			if valid {
				t.Error("expected verification to fail for tampered data")
			}
			if verifiedState != nil {
				t.Error("expected nil state for tampered data")
			}
		})
	}
}

// TestVerifyQuizState_InvalidHex tests handling of invalid hex signatures
func TestVerifyQuizState_InvalidHex(t *testing.T) {
	state := QuizState{
		QuestionIDs:  []string{"q1"},
		CurrentIndex: 0,
		Score:        0,
	}

	stateJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	tests := []struct {
		name      string
		signature string
	}{
		{
			name:      "invalid hex characters",
			signature: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		},
		{
			name:      "non-hex string",
			signature: "not-a-hex-string",
		},
		{
			name:      "odd length hex",
			signature: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifiedState, valid := verifyQuizState(string(stateJSON), tt.signature)

			if valid {
				t.Error("expected verification to fail for invalid hex")
			}
			if verifiedState != nil {
				t.Error("expected nil state for invalid hex")
			}
		})
	}
}

// TestVerifyQuizState_InvalidJSON tests handling of invalid JSON
func TestVerifyQuizState_InvalidJSON(t *testing.T) {
	state := QuizState{
		QuestionIDs:  []string{"q1", "q2"},
		CurrentIndex: 0,
		Score:        0,
	}

	// Generate a valid signature for the state
	signature, err := signQuizState(state)
	if err != nil {
		t.Fatalf("signQuizState() failed: %v", err)
	}

	tests := []struct {
		name        string
		invalidJSON string
	}{
		{
			name:        "malformed JSON",
			invalidJSON: `{this is not valid json}`,
		},
		{
			name:        "incomplete JSON",
			invalidJSON: `{"question_ids":["q1","q2"]`,
		},
		{
			name:        "wrong type",
			invalidJSON: `"not an object"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifiedState, valid := verifyQuizState(tt.invalidJSON, signature)

			if valid {
				t.Error("expected verification to fail for invalid JSON")
			}
			if verifiedState != nil {
				t.Error("expected nil state for invalid JSON")
			}
		})
	}
}

// TestQuizStateRoundTrip tests complete round-trip of sign and verify
func TestQuizStateRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		state QuizState
	}{
		{
			name: "typical state",
			state: QuizState{
				QuestionIDs:  []string{"q1", "q2", "q3", "q4", "q5"},
				CurrentIndex: 2,
				Score:        10,
			},
		},
		{
			name: "empty question IDs",
			state: QuizState{
				QuestionIDs:  []string{},
				CurrentIndex: 0,
				Score:        0,
			},
		},
		{
			name: "maximum values",
			state: QuizState{
				QuestionIDs:  []string{"q1", "q2", "q3", "q4", "q5", "q6", "q7", "q8", "q9", "q10"},
				CurrentIndex: 9,
				Score:        100,
			},
		},
		{
			name: "single question",
			state: QuizState{
				QuestionIDs:  []string{"q1"},
				CurrentIndex: 0,
				Score:        1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sign the state
			signature, err := signQuizState(tt.state)
			if err != nil {
				t.Fatalf("signQuizState() failed: %v", err)
			}

			// Serialize to JSON
			stateJSON, err := json.Marshal(tt.state)
			if err != nil {
				t.Fatalf("json.Marshal() failed: %v", err)
			}

			// Verify with signature
			verifiedState, valid := verifyQuizState(string(stateJSON), signature)

			if !valid {
				t.Error("expected valid signature after round-trip")
			}

			if verifiedState == nil {
				t.Fatal("expected non-nil state after round-trip")
			}

			// Verify all fields match
			if verifiedState.CurrentIndex != tt.state.CurrentIndex {
				t.Errorf("CurrentIndex mismatch: expected %d, got %d", tt.state.CurrentIndex, verifiedState.CurrentIndex)
			}
			if verifiedState.Score != tt.state.Score {
				t.Errorf("Score mismatch: expected %d, got %d", tt.state.Score, verifiedState.Score)
			}
			if len(verifiedState.QuestionIDs) != len(tt.state.QuestionIDs) {
				t.Errorf("QuestionIDs length mismatch: expected %d, got %d", len(tt.state.QuestionIDs), len(verifiedState.QuestionIDs))
			}
			for i := range tt.state.QuestionIDs {
				if verifiedState.QuestionIDs[i] != tt.state.QuestionIDs[i] {
					t.Errorf("QuestionIDs[%d] mismatch: expected %s, got %s", i, tt.state.QuestionIDs[i], verifiedState.QuestionIDs[i])
				}
			}
		})
	}
}
