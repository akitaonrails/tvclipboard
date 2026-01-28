package token

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

// SessionToken represents a session token with ID and timestamp
type SessionToken struct {
	ID        string    `json:"id"`
	Timestamp int64     `json:"timestamp"`
}

// TokenManager manages session tokens with encryption and expiration
type TokenManager struct {
	tokens      map[string]SessionToken
	privateKey  []byte
	timeout     time.Duration
	mu          *sync.RWMutex
	stopCleanup chan struct{}
}

// generatePrivateKey generates a 32-byte random private key
func generatePrivateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}
	return key, nil
}

// encryptToken encrypts a session token using AES-GCM
func encryptToken(token SessionToken, privateKey []byte) (string, error) {
	jsonData, err := json.Marshal(token)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(privateKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)
	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decryptToken decrypts an encrypted session token
func decryptToken(encrypted string, privateKey []byte) (SessionToken, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encrypted)
	if err != nil {
		return SessionToken{}, err
	}

	block, err := aes.NewCipher(privateKey)
	if err != nil {
		return SessionToken{}, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return SessionToken{}, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return SessionToken{}, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return SessionToken{}, err
	}

	var token SessionToken
	if err := json.Unmarshal(plaintext, &token); err != nil {
		return SessionToken{}, err
	}

	return token, nil
}

// NewTokenManager creates a new TokenManager with optional private key and timeout
func NewTokenManager(privateKeyHex string, timeoutMinutes int) *TokenManager {
	var privateKey []byte
	if privateKeyHex != "" {
		key, err := hex.DecodeString(privateKeyHex)
		if err != nil || len(key) != 32 {
			log.Printf("Invalid private key format, generating new one: %v", err)
			var genErr error
			privateKey, genErr = generatePrivateKey()
			if genErr != nil {
				log.Printf("Failed to generate new private key: %v", genErr)
				return nil
			}
		} else {
			privateKey = key
		}
	} else {
		var err error
		privateKey, err = generatePrivateKey()
		if err != nil {
			log.Printf("Failed to generate private key: %v", err)
			return nil
		}
	}

	timeout := 10 * time.Minute
	if timeoutMinutes > 0 {
		timeout = time.Duration(timeoutMinutes) * time.Minute
	}

	tm := &TokenManager{
		tokens:      make(map[string]SessionToken),
		privateKey:  privateKey,
		timeout:     timeout,
		mu:          &sync.RWMutex{},
		stopCleanup: make(chan struct{}),
	}

	tm.startCleanupRoutine()
	return tm
}

// GenerateToken creates and returns an encrypted session token
func (tm *TokenManager) GenerateToken() (string, SessionToken, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Generate 12 random bytes for ID (96 bits of entropy)
	idBytes := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, idBytes); err != nil {
		return "", SessionToken{}, fmt.Errorf("failed to generate token ID: %w", err)
	}
	tokenID := hex.EncodeToString(idBytes)

	token := SessionToken{
		ID:        tokenID,
		Timestamp: time.Now().Unix(),
	}

	tm.tokens[token.ID] = token

	encrypted, err := encryptToken(token, tm.privateKey)
	if err != nil {
		return "", SessionToken{}, err
	}

	return encrypted, token, nil
}

// ValidateToken validates an encrypted token and returns the session token
func (tm *TokenManager) ValidateToken(encrypted string) (SessionToken, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	token, err := decryptToken(encrypted, tm.privateKey)
	if err != nil {
		return SessionToken{}, fmt.Errorf("invalid token")
	}

	storedToken, ok := tm.tokens[token.ID]
	if !ok {
		return SessionToken{}, fmt.Errorf("token not found")
	}

	if time.Since(time.Unix(storedToken.Timestamp, 0)) > tm.timeout {
		return SessionToken{}, fmt.Errorf("token expired")
	}

	return storedToken, nil
}

// startCleanupRoutine starts a background routine to clean up expired tokens
func (tm *TokenManager) startCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tm.cleanupExpiredTokens()
			case <-tm.stopCleanup:
				return
			}
		}
	}()
}

// Stop stops the cleanup routine
func (tm *TokenManager) Stop() {
	close(tm.stopCleanup)
}

// cleanupExpiredTokens removes expired tokens from the map
func (tm *TokenManager) cleanupExpiredTokens() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for id, token := range tm.tokens {
		if time.Since(time.Unix(token.Timestamp, 0)) > tm.timeout {
			delete(tm.tokens, id)
			log.Printf("Cleaned up expired token: %s", id)
		}
	}
}

// Timeout returns the token timeout duration
func (tm *TokenManager) Timeout() time.Duration {
	return tm.timeout
}

// PrivateKey returns the private key (for testing only)
func (tm *TokenManager) PrivateKey() []byte {
	return tm.privateKey
}

// StoreToken stores a token in the map (for testing only)
func (tm *TokenManager) StoreToken(token SessionToken) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tokens[token.ID] = token
}

// Exports for testing
func GeneratePrivateKey() ([]byte, error) {
	return generatePrivateKey()
}

func EncryptToken(token SessionToken, privateKey []byte) (string, error) {
	return encryptToken(token, privateKey)
}

func DecryptToken(encrypted string, privateKey []byte) (SessionToken, error) {
	return decryptToken(encrypted, privateKey)
}
