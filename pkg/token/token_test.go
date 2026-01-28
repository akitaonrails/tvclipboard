package token

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestTokenGeneration tests that tokens are generated correctly
func TestTokenGeneration(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Generate a token
	encrypted, token, err := tm.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Check that encrypted token is not empty
	if encrypted == "" {
		t.Error("Encrypted token should not be empty")
	}

	// Check that token ID is valid hex (24 characters = 12 bytes)
	if len(token.ID) != 24 {
		t.Errorf("Token ID should be 24 hex characters, got %d", len(token.ID))
	}
	if _, err := hex.DecodeString(token.ID); err != nil {
		t.Errorf("Token ID should be valid hex: %v", err)
	}

	// Check that timestamp is recent (converted from Unix timestamp)
	if time.Since(time.Unix(token.Timestamp, 0)) > 5*time.Second {
		t.Error("Token timestamp should be recent")
	}
}

// TestTokenEncryptionDecryption tests that tokens can be encrypted and decrypted
func TestTokenEncryptionDecryption(t *testing.T) {
	privateKey, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Generate 12 random bytes for ID
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		t.Fatalf("Failed to generate token ID: %v", err)
	}
	tokenID := hex.EncodeToString(idBytes)

	token := SessionToken{
		ID:        tokenID,
		Timestamp: time.Now().Unix(),
	}

	// Encrypt the token
	encrypted, err := EncryptToken(token, privateKey)
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}

	// Decrypt the token
	decrypted, err := DecryptToken(encrypted, privateKey)
	if err != nil {
		t.Fatalf("Failed to decrypt token: %v", err)
	}

	// Check that decrypted token matches original
	if decrypted.ID != token.ID {
		t.Errorf("Token ID mismatch: got %s, want %s", decrypted.ID, token.ID)
	}

	// Check that timestamps match (both are Unix timestamps)
	if decrypted.Timestamp != token.Timestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decrypted.Timestamp, token.Timestamp)
	}
}

// TestTokenWithDifferentKey tests that decryption fails with wrong key
func TestTokenWithDifferentKey(t *testing.T) {
	key1, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}
	key2, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Generate 12 random bytes for ID
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		t.Fatalf("Failed to generate token ID: %v", err)
	}
	tokenID := hex.EncodeToString(idBytes)

	token := SessionToken{
		ID:        tokenID,
		Timestamp: time.Now().Unix(),
	}

	// Encrypt with key1
	encrypted, err := EncryptToken(token, key1)
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}

	// Try to decrypt with key2 (should fail)
	_, err = DecryptToken(encrypted, key2)
	if err == nil {
		t.Error("Decryption should fail with different key")
	}
}

// TestTokenValidationValid tests that valid tokens pass validation
func TestTokenValidationValid(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Generate a token
	encrypted, token, err := tm.GenerateToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate the token
	validated, err := tm.ValidateToken(encrypted)
	if err != nil {
		t.Fatalf("Token validation failed for valid token: %v", err)
	}

	// Check that validated token matches original
	if validated.ID != token.ID {
		t.Errorf("Token ID mismatch: got %s, want %s", validated.ID, token.ID)
	}
}

// TestTokenValidationInvalid tests that invalid tokens fail validation
func TestTokenValidationInvalid(t *testing.T) {
	tm := NewTokenManager("", 10)

	// Test with completely invalid string
	invalidTokens := []string{
		"",
		"invalid",
		base64.StdEncoding.EncodeToString([]byte("not a real token")),
		"YWZzaCZrZXk=", // valid base64 but not a token
	}

	for _, invalidToken := range invalidTokens {
		_, err := tm.ValidateToken(invalidToken)
		if err == nil {
			t.Errorf("Validation should fail for invalid token: %s", invalidToken)
		}
	}
}

// TestTokenValidationExpired tests that expired tokens fail validation
func TestTokenValidationExpired(t *testing.T) {
	tm := NewTokenManager("", 1) // 1 minute timeout

	// Create an expired token manually
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		t.Fatalf("Failed to generate token ID: %v", err)
	}
	token := SessionToken{
		ID:        hex.EncodeToString(idBytes),
		Timestamp: time.Now().Add(-2 * time.Minute).Unix(), // Expired
	}

	// Store the expired token
	tm.mu.Lock()
	tm.tokens[token.ID] = token
	tm.mu.Unlock()

	// Encrypt the token
	encrypted, err := EncryptToken(token, tm.privateKey)
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}

	// Try to validate (should fail)
	_, err = tm.ValidateToken(encrypted)
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

	// Create a token but don't store it
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		t.Fatalf("Failed to generate token ID: %v", err)
	}
	token := SessionToken{
		ID:        hex.EncodeToString(idBytes),
		Timestamp: time.Now().Unix(),
	}

	// Encrypt the token
	encrypted, err := EncryptToken(token, tm.privateKey)
	if err != nil {
		t.Fatalf("Failed to encrypt token: %v", err)
	}

	// Try to validate (should fail - token not in map)
	_, err = tm.ValidateToken(encrypted)
	if err == nil {
		t.Error("Validation should fail for unknown token")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error should mention 'not found': %v", err)
	}
}

// TestTokenCleanup tests that expired tokens are cleaned up
func TestTokenCleanup(t *testing.T) {
	tm := NewTokenManager("", 1) // 1 minute timeout

	// Generate some tokens
	var tokenIDs []string
	for i := 0; i < 3; i++ {
		_, token, err := tm.GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}
		tokenIDs = append(tokenIDs, token.ID)
	}

	// Manually expire one token
	tm.mu.Lock()
	expiredToken := tm.tokens[tokenIDs[0]]
	expiredToken.Timestamp = time.Now().Add(-2 * time.Minute).Unix()
	tm.tokens[tokenIDs[0]] = expiredToken
	tm.mu.Unlock()

	// Run cleanup
	tm.cleanupExpiredTokens()

	// Check that expired token was removed
	tm.mu.RLock()
	_, exists := tm.tokens[tokenIDs[0]]
	tm.mu.RUnlock()

	if exists {
		t.Error("Expired token should be removed from map")
	}

	// Check that other tokens still exist
	for i := 1; i < len(tokenIDs); i++ {
		tm.mu.RLock()
		_, exists := tm.tokens[tokenIDs[i]]
		tm.mu.RUnlock()

		if !exists {
			t.Errorf("Valid token %d should still exist", i)
		}
	}
}

// TestPrivateKeyGeneration tests that private keys are generated correctly
func TestPrivateKeyGeneration(t *testing.T) {
	key1, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}
	key2, err := GeneratePrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Keys should be different
	if bytes.Equal(key1, key2) {
		t.Error("Generated keys should be different")
	}

	// Keys should be 32 bytes
	if len(key1) != 32 {
		t.Errorf("Key should be 32 bytes, got %d", len(key1))
	}

	if len(key2) != 32 {
		t.Errorf("Key should be 32 bytes, got %d", len(key2))
	}
}

// TestPrivateKeyFromEnv tests that private keys can be set from hex string
func TestPrivateKeyFromEnv(t *testing.T) {
	hexKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	tm := NewTokenManager(hexKey, 10)

	// Check that private key matches
	expectedKey, _ := hex.DecodeString(hexKey)
	if !bytes.Equal(tm.privateKey, expectedKey) {
		t.Error("Private key should match provided hex string")
	}
}

// TestPrivateKeyInvalidHex tests that invalid hex generates new key
func TestPrivateKeyInvalidHex(t *testing.T) {
	tm1 := NewTokenManager("invalid-hex", 10)
	tm2 := NewTokenManager("", 10)

	// Invalid hex should generate new random key
	if bytes.Equal(tm1.privateKey, tm2.privateKey) {
		t.Error("Invalid hex should generate random key, but keys should differ")
	}
}

// TestGeneratePrivateKey tests that private key is 32 bytes and cryptographically random
func TestGeneratePrivateKey(t *testing.T) {
	// Generate multiple keys and verify they're different
	var keys [][]byte
	for i := 0; i < 100; i++ {
		key, err := GeneratePrivateKey()
		if err != nil {
			t.Fatalf("Failed to generate private key: %v", err)
		}
		keys = append(keys, key)

		// Check length
		if len(key) != 32 {
			t.Errorf("Key should be 32 bytes, got %d", len(key))
		}

		// Check all bits are set (some entropy)
		allZero := true
		for _, b := range key {
			if b != 0 {
				allZero = false
				break
			}
		}
		if allZero {
			t.Errorf("Key %d should not be all zeros", i)
		}
	}

	// Check for duplicates (highly unlikely with 100 keys)
	seen := make(map[string]bool)
	for _, key := range keys {
		keyStr := string(key)
		if seen[keyStr] {
			t.Error("Generated keys should be unique")
		}
		seen[keyStr] = true
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

	var encryptedTokens []string
	var tokens []SessionToken

	// Generate 10 tokens
	for i := 0; i < 10; i++ {
		encrypted, token, err := tm.GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token %d: %v", i, err)
		}

		encryptedTokens = append(encryptedTokens, encrypted)
		tokens = append(tokens, token)
	}

	// Validate all tokens
	for i, encrypted := range encryptedTokens {
		validated, err := tm.ValidateToken(encrypted)
		if err != nil {
			t.Errorf("Token %d validation failed: %v", i, err)
		}

		if validated.ID != tokens[i].ID {
			t.Errorf("Token %d ID mismatch", i)
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

// TestTokenJSONEncoding tests that tokens can be properly JSON encoded/decoded
func TestTokenJSONEncoding(t *testing.T) {
	idBytes := make([]byte, 12)
	if _, err := rand.Read(idBytes); err != nil {
		t.Fatalf("Failed to generate token ID: %v", err)
	}
	token := SessionToken{
		ID:        hex.EncodeToString(idBytes),
		Timestamp: time.Now().Unix(),
	}

	// Encode to JSON
	jsonData, err := json.Marshal(token)
	if err != nil {
		t.Fatalf("Failed to marshal token: %v", err)
	}

	// Decode from JSON
	var decoded SessionToken
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal token: %v", err)
	}

	// Verify fields match
	if decoded.ID != token.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, token.ID)
	}

	// Timestamps should match exactly (both are Unix timestamps)
	if decoded.Timestamp != token.Timestamp {
		t.Errorf("Timestamp mismatch: got %d, want %d", decoded.Timestamp, token.Timestamp)
	}
}
