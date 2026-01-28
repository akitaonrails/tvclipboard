package token

import (
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"time"
)

const (
	// TokenLength is the number of characters in a token ID
	TokenLength = 8
	// MaxTokens is the hard limit for in-memory token storage
	MaxTokens = 10000
)

// SessionToken represents a token with ID and timestamp
type SessionToken struct {
	ID        string
	Timestamp int64
}

// TokenManager manages session tokens with in-memory storage and size limits
type TokenManager struct {
	tokens      map[string]int64 // token ID → timestamp
	tokenOrder  []string          // FIFO order for rotation
	timeout     time.Duration
	maxTokens   int
	mu          *sync.RWMutex
}

// base62 characters for generating short alphanumeric IDs
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// generateRandomID generates a short alphanumeric ID
func generateRandomID() (string, error) {
	b := make([]byte, TokenLength)
	for i := range b {
		// Generate random byte and use modulo to get base62 index
		val, err := rand.Prime(rand.Reader, 8)
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		b[i] = base62Chars[int(val.Int64())%len(base62Chars)]
	}
	return string(b), nil
}

// NewTokenManager creates a new TokenManager with timeout and size limits
func NewTokenManager(privateKeyHex string, timeoutMinutes int) *TokenManager {
	timeout := 10 * time.Minute
	if timeoutMinutes > 0 {
		timeout = time.Duration(timeoutMinutes) * time.Minute
	}

	tm := &TokenManager{
		tokens:     make(map[string]int64),
		tokenOrder: make([]string, 0, MaxTokens),
		timeout:    timeout,
		maxTokens:  MaxTokens,
		mu:         &sync.RWMutex{},
	}

	return tm
}

// GenerateToken creates and returns a short session token ID
func (tm *TokenManager) GenerateToken() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Generate random ID and check for collision
	var tokenID string
	var err error
	maxAttempts := 100

	for i := 0; i < maxAttempts; i++ {
		tokenID, err = generateRandomID()
		if err != nil {
			return "", err
		}
		// Check if ID already exists
		if _, exists := tm.tokens[tokenID]; !exists {
			break // Found a unique ID
		}
	}

	if tokenID == "" {
		return "", fmt.Errorf("failed to generate unique token after %d attempts", maxAttempts)
	}

	// Add token to map and order list
	now := time.Now().Unix()
	tm.tokens[tokenID] = now
	tm.tokenOrder = append(tm.tokenOrder, tokenID)

	// Enforce max tokens limit by removing oldest entries
	for len(tm.tokens) > tm.maxTokens {
		oldestID := tm.tokenOrder[0]
		delete(tm.tokens, oldestID)
		// Remove from order list (shift slice)
		tm.tokenOrder = tm.tokenOrder[1:]
		log.Printf("Rotated out oldest token due to max limit: %s", oldestID)
	}

	return tokenID, nil
}

// ValidateToken validates a token ID and returns if it's still valid
func (tm *TokenManager) ValidateToken(tokenID string) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	timestamp, exists := tm.tokens[tokenID]
	if !exists {
		return fmt.Errorf("token not found")
	}

	if time.Since(time.Unix(timestamp, 0)) > tm.timeout {
		return fmt.Errorf("token expired")
	}

	return nil
}

// Timeout returns the token timeout duration
func (tm *TokenManager) Timeout() time.Duration {
	return tm.timeout
}

// StoreToken stores a token in map (for testing only)
func (tm *TokenManager) StoreToken(token SessionToken) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tokens[token.ID] = token.Timestamp
	tm.tokenOrder = append(tm.tokenOrder, token.ID)
}

// GetTokens returns the current token count (for testing)
func (tm *TokenManager) GetTokens() map[string]int64 {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	// Return a copy
	copy := make(map[string]int64)
	for k, v := range tm.tokens {
		copy[k] = v
	}
	return copy
}

// TokenCount returns the number of active tokens (for testing)
func (tm *TokenManager) TokenCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.tokens)
}
