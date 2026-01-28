package token

import (
	"strings"
	"testing"
	"time"
)

// TestTokenGeneration tests that tokens are generated correctly
func TestTokenGeneration(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Generate a token
	tokenID, err := tm.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Check that token is not empty
	if tokenID == "" {
		t.Error("Token should not be empty")
	}

	// Check that token is 8 characters
	if len(tokenID) != TokenLength {
		t.Errorf("Token should be %d characters, got %d", TokenLength, len(tokenID))
	}

	// Check that token only contains alphanumeric characters
	for _, r := range tokenID {
		if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			t.Errorf("Token should only contain alphanumeric characters, found: %c", r)
		}
	}

	// Check that token was stored in map
	tm.mu.RLock()
	_, exists := tm.tokens[tokenID]
	tm.mu.RUnlock()

	if !exists {
		t.Error("Token should be stored in map")
	}
}

// TestTokenValidationValid tests that valid tokens pass validation
func TestTokenValidationValid(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Generate a token
	tokenID, err := tm.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate the token
	err = tm.ValidateToken(tokenID)
	if err != nil {
		t.Fatalf("Token validation failed for valid token: %v", err)
	}
}

// TestTokenValidationInvalid tests that invalid tokens fail validation
func TestTokenValidationInvalid(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Test with invalid strings
	invalidTokens := []string{
		"",
		"invalid",
		"abc123xyz", // valid format but not stored
		"1234567890", // too long
	}

	for _, invalidToken := range invalidTokens {
		err := tm.ValidateToken(invalidToken)
		if err == nil {
			t.Errorf("Validation should fail for invalid token: %s", invalidToken)
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Error should mention 'not found': %v", err)
		}
	}
}

// TestTokenValidationExpired tests that expired tokens fail validation
func TestTokenValidationExpired(t *testing.T) {
	tm := NewTokenManager("", 1) // 1 minute timeout

	// Generate a token
	tokenID, err := tm.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Manually expire the token by setting its timestamp
	tm.mu.Lock()
	tm.tokens[tokenID] = time.Now().Add(-2 * time.Minute).Unix()
	tm.mu.Unlock()

	// Try to validate (should fail)
	err = tm.ValidateToken(tokenID)
	if err == nil {
		t.Error("Validation should fail for expired token")
	}

	if !strings.Contains(err.Error(), "expired") {
		t.Errorf("Error should mention expiration: %v", err)
	}
}

// TestTokenNotFound tests that unknown tokens fail validation
func TestTokenNotFound(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Generate a token (will be stored)
	tokenID, err := tm.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Remove token from map (simulate unknown token)
	tm.mu.Lock()
	delete(tm.tokens, tokenID)
	tm.mu.Unlock()

	// Try to validate (should fail - token not in map)
	err = tm.ValidateToken(tokenID)
	if err == nil {
		t.Error("Validation should fail for unknown token")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found': %v", err)
	}
}

// TestTokenCleanup tests that FIFO limit removes oldest tokens
func TestTokenCleanup(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Generate some tokens
	var tokenIDs []string
	for i := 0; i < 5; i++ {
		tokenID, err := tm.GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
		tokenIDs = append(tokenIDs, tokenID)
	}

	// Check that all 5 tokens exist
	tm.mu.RLock()
	_, exists := tm.tokens[tokenIDs[0]]
	tm.mu.RUnlock()

	if !exists {
		t.Error("First token should exist")
	}

	// Generate tokens up to limit
	for i := 0; i < MaxTokens; i++ {
		tm.GenerateToken()
	}

	// The first token should have been rotated out due to FIFO limit
	tm.mu.RLock()
	_, exists = tm.tokens[tokenIDs[0]]
	tokenCount := len(tm.tokens)
	tm.mu.RUnlock()

	if exists {
		t.Error("Oldest token should be removed due to FIFO limit")
	}

	if tokenCount > MaxTokens {
		t.Errorf("Token count should be at most %d, got %d", MaxTokens, tokenCount)
	}
}

// TestTokenManagerTimeout tests that token timeout is correctly set
func TestTokenManagerTimeout(t *testing.T) {
	tests := []struct {
		minutes    int
		wantTimeout time.Duration
	}{
		{5, 5 * time.Minute},
		{10, 10 * time.Minute},
		{15, 15 * time.Minute},
		{60, 60 * time.Minute},
		{0, 10 * time.Minute},  // Default
		{-5, 10 * time.Minute}, // Default for negative
	}

	for _, tt := range tests {
		tm := NewTokenManager("", tt.minutes)
		if tm.timeout != tt.wantTimeout {
			t.Errorf("NewTokenManager(%d) timeout = %v, want %v", tt.minutes, tm.timeout, tt.wantTimeout)
		}
	}
}

// TestMultipleValidTokens tests that multiple tokens can be generated and validated
func TestMultipleValidTokens(t *testing.T) {
	tm := NewTokenManager("", 10)

	var tokenIDs []string

	// Generate 10 tokens
	for i := 0; i < 10; i++ {
		tokenID, err := tm.GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token %d: %v", i, err)
		}

		tokenIDs = append(tokenIDs, tokenID)
	}

	// Validate all tokens
	for i, tokenID := range tokenIDs {
		err := tm.ValidateToken(tokenID)
		if err != nil {
			t.Errorf("Token %d validation failed: %v", i, err)
		}
	}

	// Check that all tokens are stored in map
	tm.mu.RLock()
	storedCount := len(tm.tokens)
	tm.mu.RUnlock()

	if storedCount != 10 {
		t.Errorf("Expected 10 tokens in map, got %d", storedCount)
	}
}

// TestTokenUniqueness tests that generated tokens are unique
func TestTokenUniqueness(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Generate many tokens
	tokens := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		tokenID, err := tm.GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token %d: %v", i, err)
		}

		if tokens[tokenID] {
			t.Errorf("Generated duplicate token: %s", tokenID)
		}
		tokens[tokenID] = true
	}

	// Check that we have 1000 unique tokens
	if len(tokens) != 1000 {
		t.Errorf("Expected 1000 unique tokens, got %d", len(tokens))
	}
}

// TestTokenLimit tests that the max token limit is enforced
func TestTokenLimit(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Store some tokens manually (bypassing GenerateToken's limit check)
	tm.mu.Lock()
	for i := 0; i < MaxTokens + 5; i++ {
		tokenID := "T" + string(rune('a'+i%26))
		tm.tokens[tokenID] = time.Now().Unix()
		tm.tokenOrder = append(tm.tokenOrder, tokenID)
	}
	tm.mu.Unlock()

	// Generate a token - should enforce limit and remove oldest
	_, err := tm.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Check that the number of tokens is at most MaxTokens
	tm.mu.RLock()
	tokenCount := len(tm.tokens)
	tm.mu.RUnlock()

	if tokenCount > MaxTokens {
		t.Errorf("Expected at most %d tokens, got %d", MaxTokens, tokenCount)
	}
}

// TestTimeout tests that Timeout returns the configured timeout
func TestTimeout(t *testing.T) {
	tests := []struct {
		minutes    int
		wantTimeout time.Duration
	}{
		{5, 5 * time.Minute},
		{10, 10 * time.Minute},
		{0, 10 * time.Minute},  // Default
		{-5, 10 * time.Minute}, // Default for negative
	}

	for _, tt := range tests {
		tm := NewTokenManager("", tt.minutes)
		got := tm.Timeout()
		if got != tt.wantTimeout {
			t.Errorf("Timeout() = %v, want %v", got, tt.wantTimeout)
		}
	}
}

// TestStoreToken tests direct token storage
func TestStoreToken(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Create a token manually
	token := SessionToken{
		ID:        "testtoken123",
		Timestamp: time.Now().Unix(),
	}

	// Store it
	tm.StoreToken(token)

	// Verify it's in the map
	tm.mu.RLock()
	stored, exists := tm.tokens[token.ID]
	tm.mu.RUnlock()

	if !exists {
		t.Error("Token should be stored")
	}
	if stored != token.Timestamp {
		t.Error("Token timestamp should match")
	}
}

// TestGetTokens tests retrieving all tokens
func TestGetTokens(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Generate some tokens
	var expectedIDs []string
	for i := 0; i < 5; i++ {
		tokenID, err := tm.GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
		expectedIDs = append(expectedIDs, tokenID)
	}

	// Get all tokens (returns map[string]int64)
	tokens := tm.GetTokens()

	// Verify count
	if len(tokens) != len(expectedIDs) {
		t.Errorf("Got %d tokens, want %d", len(tokens), len(expectedIDs))
	}

	// Verify all IDs are present
	for _, id := range expectedIDs {
		if _, exists := tokens[id]; !exists {
			t.Errorf("Token ID %s not found in GetTokens result", id)
		}
	}
}

// TestTokenCount tests counting tokens
func TestTokenCount(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Initially should be 0
	if count := tm.TokenCount(); count != 0 {
		t.Errorf("Initial count should be 0, got %d", count)
	}

	// Add some tokens
	for i := 0; i < 5; i++ {
		tm.GenerateToken()
	}

	// Should be 5
	if count := tm.TokenCount(); count != 5 {
		t.Errorf("Count should be 5, got %d", count)
	}
}

