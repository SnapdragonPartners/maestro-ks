package main

import (
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
