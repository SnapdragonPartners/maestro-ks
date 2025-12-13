package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// TestLeaderboardEntry_JSONSerialization tests marshaling and unmarshaling LeaderboardEntry
func TestLeaderboardEntry_JSONSerialization(t *testing.T) {
	// Create a test entry with a specific timestamp
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	original := LeaderboardEntry{
		Name:  "Alice",
		Score: 95,
		Total: 100,
		When:  testTime,
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	// Unmarshal back
	var unmarshaled LeaderboardEntry
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}

	// Verify all fields match
	if unmarshaled.Name != original.Name {
		t.Errorf("Name mismatch: expected %s, got %s", original.Name, unmarshaled.Name)
	}
	if unmarshaled.Score != original.Score {
		t.Errorf("Score mismatch: expected %d, got %d", original.Score, unmarshaled.Score)
	}
	if unmarshaled.Total != original.Total {
		t.Errorf("Total mismatch: expected %d, got %d", original.Total, unmarshaled.Total)
	}
	if !unmarshaled.When.Equal(original.When) {
		t.Errorf("When mismatch: expected %v, got %v", original.When, unmarshaled.When)
	}
}

// TestLoadLeaderboard_FileNotExists tests loading when file doesn't exist
func TestLoadLeaderboard_FileNotExists(t *testing.T) {
	// Change to temporary directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Reset manager
	leaderboardManager = LeaderboardManager{}

	// Load leaderboard (file doesn't exist)
	err := loadLeaderboard()
	if err != nil {
		t.Fatalf("loadLeaderboard() failed: %v", err)
	}

	// Verify empty array is created
	if len(leaderboardManager.entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(leaderboardManager.entries))
	}

	// Verify file was created with empty array
	data, err := os.ReadFile(leaderboardFilename)
	if err != nil {
		t.Fatalf("failed to read created file: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("expected file content '[]', got '%s'", string(data))
	}
}

// TestLoadLeaderboard_ValidFile tests loading with valid JSON entries
func TestLoadLeaderboard_ValidFile(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create valid JSON file
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	entries := []LeaderboardEntry{
		{Name: "Alice", Score: 100, Total: 100, When: testTime},
		{Name: "Bob", Score: 90, Total: 100, When: testTime.Add(time.Hour)},
	}

	data, _ := json.Marshal(entries)
	if err := os.WriteFile(leaderboardFilename, data, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Reset manager
	leaderboardManager = LeaderboardManager{}

	// Load leaderboard
	err := loadLeaderboard()
	if err != nil {
		t.Fatalf("loadLeaderboard() failed: %v", err)
	}

	// Verify entries loaded correctly
	if len(leaderboardManager.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(leaderboardManager.entries))
	}

	// Verify first entry
	if leaderboardManager.entries[0].Name != "Alice" {
		t.Errorf("expected Name 'Alice', got '%s'", leaderboardManager.entries[0].Name)
	}
	if leaderboardManager.entries[0].Score != 100 {
		t.Errorf("expected Score 100, got %d", leaderboardManager.entries[0].Score)
	}
	if leaderboardManager.entries[0].Total != 100 {
		t.Errorf("expected Total 100, got %d", leaderboardManager.entries[0].Total)
	}

	// Verify second entry
	if leaderboardManager.entries[1].Name != "Bob" {
		t.Errorf("expected Name 'Bob', got '%s'", leaderboardManager.entries[1].Name)
	}
}

// TestLoadLeaderboard_InvalidJSON tests with malformed JSON
func TestLoadLeaderboard_InvalidJSON(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Create invalid JSON file
	invalidJSON := `{this is not valid json}`
	if err := os.WriteFile(leaderboardFilename, []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Reset manager
	leaderboardManager = LeaderboardManager{}

	// Load leaderboard
	err := loadLeaderboard()
	if err == nil {
		t.Error("expected error for invalid JSON, got none")
	}

	if !contains(err.Error(), "failed to parse leaderboard JSON") {
		t.Errorf("expected error message about parsing JSON, got: %v", err)
	}
}

// TestSaveScore_SingleEntry tests saving a single score
func TestSaveScore_SingleEntry(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Reset manager
	leaderboardManager = LeaderboardManager{entries: []LeaderboardEntry{}}

	// Save a score
	before := time.Now()
	err := saveScore("Alice", 95, 100)
	after := time.Now()

	if err != nil {
		t.Fatalf("saveScore() failed: %v", err)
	}

	// Verify entry was added
	if len(leaderboardManager.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(leaderboardManager.entries))
	}

	entry := leaderboardManager.entries[0]
	if entry.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got '%s'", entry.Name)
	}
	if entry.Score != 95 {
		t.Errorf("expected Score 95, got %d", entry.Score)
	}
	if entry.Total != 100 {
		t.Errorf("expected Total 100, got %d", entry.Total)
	}

	// Verify timestamp is set and reasonable
	if entry.When.Before(before) || entry.When.After(after) {
		t.Errorf("timestamp %v is not between %v and %v", entry.When, before, after)
	}

	// Verify file was created
	data, err := os.ReadFile(leaderboardFilename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var fileEntries []LeaderboardEntry
	if err := json.Unmarshal(data, &fileEntries); err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	if len(fileEntries) != 1 {
		t.Errorf("expected 1 entry in file, got %d", len(fileEntries))
	}
}

// TestSaveScore_Sorting tests that entries are sorted correctly
func TestSaveScore_Sorting(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Reset manager
	leaderboardManager = LeaderboardManager{entries: []LeaderboardEntry{}}

	// Add entries with various scores and dates
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Manually add entries to test sorting
	leaderboardManager.entries = []LeaderboardEntry{
		{Name: "Alice", Score: 100, Total: 100, When: baseTime},
		{Name: "Bob", Score: 100, Total: 100, When: baseTime.Add(24 * time.Hour)},
		{Name: "Carol", Score: 95, Total: 100, When: baseTime},
		{Name: "Dave", Score: 105, Total: 100, When: baseTime.Add(48 * time.Hour)},
	}

	// Call saveScore to trigger sorting
	if err := saveScore("Eve", 98, 100); err != nil {
		t.Fatalf("saveScore() failed: %v", err)
	}

	// Expected order after sorting:
	// 1. Dave (105) - highest score
	// 2. Alice (100, earlier date)
	// 3. Bob (100, later date)
	// 4. Eve (98)
	// 5. Carol (95) - lowest score

	if len(leaderboardManager.entries) != 5 {
		t.Fatalf("expected 5 entries, got %d", len(leaderboardManager.entries))
	}

	// Verify order
	if leaderboardManager.entries[0].Name != "Dave" {
		t.Errorf("expected Dave first (highest score), got %s", leaderboardManager.entries[0].Name)
	}
	if leaderboardManager.entries[1].Name != "Alice" {
		t.Errorf("expected Alice second (100, earlier date), got %s", leaderboardManager.entries[1].Name)
	}
	if leaderboardManager.entries[2].Name != "Bob" {
		t.Errorf("expected Bob third (100, later date), got %s", leaderboardManager.entries[2].Name)
	}
	if leaderboardManager.entries[3].Name != "Eve" {
		t.Errorf("expected Eve fourth (98), got %s", leaderboardManager.entries[3].Name)
	}
	if leaderboardManager.entries[4].Name != "Carol" {
		t.Errorf("expected Carol fifth (lowest score), got %s", leaderboardManager.entries[4].Name)
	}
}

// TestSaveScore_Truncation tests that only top 20 entries are kept
func TestSaveScore_Truncation(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Reset manager
	leaderboardManager = LeaderboardManager{entries: []LeaderboardEntry{}}

	// Add 25 entries (exceeding MaxLeaderboardSize of 20)
	for i := 0; i < 25; i++ {
		// Use descending scores so we know which should be kept
		err := saveScore(fmt.Sprintf("Player%d", i), 100-i, 100)
		if err != nil {
			t.Fatalf("saveScore() failed for player %d: %v", i, err)
		}
	}

	// Verify only top 20 remain
	if len(leaderboardManager.entries) != MaxLeaderboardSize {
		t.Errorf("expected %d entries, got %d", MaxLeaderboardSize, len(leaderboardManager.entries))
	}

	// Verify highest scores are kept (Player0 through Player19)
	for i := 0; i < MaxLeaderboardSize; i++ {
		expectedName := fmt.Sprintf("Player%d", i)
		if leaderboardManager.entries[i].Name != expectedName {
			t.Errorf("expected entry %d to be %s, got %s", i, expectedName, leaderboardManager.entries[i].Name)
		}
	}

	// Verify lowest scores are removed (Player20-24 should not be present)
	for _, entry := range leaderboardManager.entries {
		if entry.Name == "Player20" || entry.Name == "Player24" {
			t.Errorf("entry %s should have been truncated", entry.Name)
		}
	}

	// Verify file only contains 20 entries
	data, err := os.ReadFile(leaderboardFilename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var fileEntries []LeaderboardEntry
	if err := json.Unmarshal(data, &fileEntries); err != nil {
		t.Fatalf("failed to parse file: %v", err)
	}

	if len(fileEntries) != MaxLeaderboardSize {
		t.Errorf("expected %d entries in file, got %d", MaxLeaderboardSize, len(fileEntries))
	}
}

// TestSaveScore_Persistence tests that scores persist across manager instances
func TestSaveScore_Persistence(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Reset manager
	leaderboardManager = LeaderboardManager{entries: []LeaderboardEntry{}}

	// Save some scores
	testTime := time.Now()
	if err := saveScore("Alice", 100, 100); err != nil {
		t.Fatalf("saveScore() failed: %v", err)
	}
	if err := saveScore("Bob", 90, 100); err != nil {
		t.Fatalf("saveScore() failed: %v", err)
	}

	// Verify initial entries
	if len(leaderboardManager.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(leaderboardManager.entries))
	}

	// Create a new manager instance and load
	leaderboardManager = LeaderboardManager{}
	if err := loadLeaderboard(); err != nil {
		t.Fatalf("loadLeaderboard() failed: %v", err)
	}

	// Verify entries persisted correctly
	if len(leaderboardManager.entries) != 2 {
		t.Fatalf("expected 2 persisted entries, got %d", len(leaderboardManager.entries))
	}

	if leaderboardManager.entries[0].Name != "Alice" {
		t.Errorf("expected first entry 'Alice', got '%s'", leaderboardManager.entries[0].Name)
	}
	if leaderboardManager.entries[0].Score != 100 {
		t.Errorf("expected first entry score 100, got %d", leaderboardManager.entries[0].Score)
	}

	if leaderboardManager.entries[1].Name != "Bob" {
		t.Errorf("expected second entry 'Bob', got '%s'", leaderboardManager.entries[1].Name)
	}
	if leaderboardManager.entries[1].Score != 90 {
		t.Errorf("expected second entry score 90, got %d", leaderboardManager.entries[1].Score)
	}

	// Verify timestamps are preserved
	if leaderboardManager.entries[0].When.Before(testTime.Add(-time.Minute)) {
		t.Errorf("timestamp seems too old: %v", leaderboardManager.entries[0].When)
	}
}

// TestSaveScore_Concurrent tests concurrent writes are handled safely
func TestSaveScore_Concurrent(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	// Reset manager
	leaderboardManager = LeaderboardManager{entries: []LeaderboardEntry{}}

	// Launch 50 goroutines calling saveScore simultaneously
	var wg sync.WaitGroup
	numGoroutines := 50
	errorsChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := saveScore(fmt.Sprintf("Player%d", idx), idx*10, 100)
			if err != nil {
				errorsChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errorsChan)

	// Check for any errors
	for err := range errorsChan {
		t.Errorf("concurrent saveScore() failed: %v", err)
	}

	// Verify all writes succeeded
	// Should have min(50, MaxLeaderboardSize) entries
	expectedCount := numGoroutines
	if expectedCount > MaxLeaderboardSize {
		expectedCount = MaxLeaderboardSize
	}

	if len(leaderboardManager.entries) != expectedCount {
		t.Errorf("expected %d entries, got %d", expectedCount, len(leaderboardManager.entries))
	}

	// Verify no data corruption by checking entries are valid
	for i, entry := range leaderboardManager.entries {
		if entry.Name == "" {
			t.Errorf("entry %d has empty name", i)
		}
		if entry.Score < 0 || entry.Score > 500 {
			t.Errorf("entry %d has invalid score: %d", i, entry.Score)
		}
		if entry.Total != 100 {
			t.Errorf("entry %d has invalid total: %d", i, entry.Total)
		}
		if entry.When.IsZero() {
			t.Errorf("entry %d has zero timestamp", i)
		}
	}

	// Verify entries are properly sorted (score descending)
	for i := 0; i < len(leaderboardManager.entries)-1; i++ {
		if leaderboardManager.entries[i].Score < leaderboardManager.entries[i+1].Score {
			t.Errorf("entries not properly sorted at index %d: %d < %d",
				i, leaderboardManager.entries[i].Score, leaderboardManager.entries[i+1].Score)
		}
	}
}

// TestGetLeaderboard_ReturnsCopy tests that getLeaderboard returns a copy
func TestGetLeaderboard_ReturnsCopy(t *testing.T) {
	// Reset manager with test data
	leaderboardManager = LeaderboardManager{
		entries: []LeaderboardEntry{
			{Name: "Alice", Score: 100, Total: 100, When: time.Now()},
			{Name: "Bob", Score: 90, Total: 100, When: time.Now()},
		},
	}

	// Get leaderboard
	copied := getLeaderboard()

	// Modify the returned slice
	if len(copied) > 0 {
		copied[0].Name = "Modified"
		copied[0].Score = 999
	}

	// Verify original is unchanged
	if leaderboardManager.entries[0].Name != "Alice" {
		t.Errorf("original was modified: expected 'Alice', got '%s'", leaderboardManager.entries[0].Name)
	}
	if leaderboardManager.entries[0].Score != 100 {
		t.Errorf("original was modified: expected 100, got %d", leaderboardManager.entries[0].Score)
	}
}

// TestGetLeaderboard_MaxSize tests that getLeaderboard returns up to 20 entries
func TestGetLeaderboard_MaxSize(t *testing.T) {
	// Reset manager with more than 20 entries
	entries := make([]LeaderboardEntry, 25)
	for i := 0; i < 25; i++ {
		entries[i] = LeaderboardEntry{
			Name:  fmt.Sprintf("Player%d", i),
			Score: 100 - i,
			Total: 100,
			When:  time.Now(),
		}
	}
	leaderboardManager = LeaderboardManager{entries: entries}

	// Get leaderboard
	result := getLeaderboard()

	// Verify it returns all entries (the manager should have already truncated to 20)
	// But in this test we're bypassing saveScore, so it has 25
	if len(result) != 25 {
		t.Errorf("expected 25 entries, got %d", len(result))
	}

	// Note: The truncation happens in saveScore, not getLeaderboard
	// getLeaderboard just returns a copy of whatever is in the manager
}
