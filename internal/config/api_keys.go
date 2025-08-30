package config

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// APIKey represents an API key with metadata
type APIKey struct {
	Key        string
	Platform   string
	IsActive   bool
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	LastUsed   *time.Time
	UsageCount int64
}

// APIKeyManager manages API keys with rotation support
type APIKeyManager struct {
	keys   map[string][]*APIKey // platform -> keys
	mutex  sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager() *APIKeyManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &APIKeyManager{
		keys:   make(map[string][]*APIKey),
		ctx:    ctx,
		cancel: cancel,
	}
}

// AddKey adds a new API key for a platform
func (akm *APIKeyManager) AddKey(platform, key string, expiresAt *time.Time) error {
	akm.mutex.Lock()
	defer akm.mutex.Unlock()

	apiKey := &APIKey{
		Key:       key,
		Platform:  platform,
		IsActive:  true,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	if akm.keys[platform] == nil {
		akm.keys[platform] = make([]*APIKey, 0)
	}

	akm.keys[platform] = append(akm.keys[platform], apiKey)
	return nil
}

// GetActiveKey returns an active API key for a platform
func (akm *APIKeyManager) GetActiveKey(platform string) (*APIKey, error) {
	akm.mutex.RLock()
	defer akm.mutex.RUnlock()

	keys, exists := akm.keys[platform]
	if !exists || len(keys) == 0 {
		return nil, fmt.Errorf("no API keys found for platform: %s", platform)
	}

	// Find the first active, non-expired key
	for _, key := range keys {
		if key.IsActive && !akm.isExpired(key) {
			akm.updateUsage(key)
			return key, nil
		}
	}

	return nil, fmt.Errorf("no active API keys found for platform: %s", platform)
}

// GetNextKey returns the next available API key for a platform
func (akm *APIKeyManager) GetNextKey(platform string) (*APIKey, error) {
	akm.mutex.RLock()
	defer akm.mutex.RUnlock()

	keys, exists := akm.keys[platform]
	if !exists || len(keys) == 0 {
		return nil, fmt.Errorf("no API keys found for platform: %s", platform)
	}

	// Find the next active, non-expired key with lowest usage
	var bestKey *APIKey
	var lowestUsage int64 = -1

	for _, key := range keys {
		if key.IsActive && !akm.isExpired(key) {
			if lowestUsage == -1 || key.UsageCount < lowestUsage {
				bestKey = key
				lowestUsage = key.UsageCount
			}
		}
	}

	if bestKey == nil {
		return nil, fmt.Errorf("no active API keys found for platform: %s", platform)
	}

	akm.updateUsage(bestKey)
	return bestKey, nil
}

// RotateKeys rotates API keys for a platform
func (akm *APIKeyManager) RotateKeys(platform string) error {
	akm.mutex.Lock()
	defer akm.mutex.Unlock()

	keys, exists := akm.keys[platform]
	if !exists || len(keys) == 0 {
		return fmt.Errorf("no API keys found for platform: %s", platform)
	}

	// Deactivate all current keys
	for _, key := range keys {
		key.IsActive = false
	}

	// Activate the next key in rotation
	for i, key := range keys {
		if !akm.isExpired(key) {
			keys[i].IsActive = true
			break
		}
	}

	return nil
}

// DeactivateKey deactivates a specific API key
func (akm *APIKeyManager) DeactivateKey(platform, keyValue string) error {
	akm.mutex.Lock()
	defer akm.mutex.Unlock()

	keys, exists := akm.keys[platform]
	if !exists {
		return fmt.Errorf("no API keys found for platform: %s", platform)
	}

	for _, key := range keys {
		if key.Key == keyValue {
			key.IsActive = false
			return nil
		}
	}

	return fmt.Errorf("API key not found for platform: %s", platform)
}

// GetKeyStats returns statistics about API keys
func (akm *APIKeyManager) GetKeyStats(platform string) map[string]interface{} {
	akm.mutex.RLock()
	defer akm.mutex.RUnlock()

	keys, exists := akm.keys[platform]
	if !exists {
		return map[string]interface{}{
			"total_keys":   0,
			"active_keys":  0,
			"expired_keys": 0,
			"total_usage":  0,
		}
	}

	stats := map[string]interface{}{
		"total_keys":   len(keys),
		"active_keys":  0,
		"expired_keys": 0,
		"total_usage":  int64(0),
	}

	for _, key := range keys {
		if key.IsActive {
			stats["active_keys"] = stats["active_keys"].(int) + 1
		}
		if akm.isExpired(key) {
			stats["expired_keys"] = stats["expired_keys"].(int) + 1
		}
		stats["total_usage"] = stats["total_usage"].(int64) + key.UsageCount
	}

	return stats
}

// CleanupExpiredKeys removes expired keys
func (akm *APIKeyManager) CleanupExpiredKeys() {
	akm.mutex.Lock()
	defer akm.mutex.Unlock()

	for platform, keys := range akm.keys {
		var activeKeys []*APIKey
		for _, key := range keys {
			if !akm.isExpired(key) {
				activeKeys = append(activeKeys, key)
			}
		}
		akm.keys[platform] = activeKeys
	}
}

// StartKeyRotation starts automatic key rotation
func (akm *APIKeyManager) StartKeyRotation(rotationInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(rotationInterval)
		defer ticker.Stop()

		for {
			select {
			case <-akm.ctx.Done():
				return
			case <-ticker.C:
				akm.rotateAllKeys()
			}
		}
	}()
}

// Stop stops the key rotation
func (akm *APIKeyManager) Stop() {
	akm.cancel()
}

// isExpired checks if a key is expired
func (akm *APIKeyManager) isExpired(key *APIKey) bool {
	if key.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*key.ExpiresAt)
}

// updateUsage updates the usage statistics for a key
func (akm *APIKeyManager) updateUsage(key *APIKey) {
	now := time.Now()
	key.LastUsed = &now
	key.UsageCount++
}

// rotateAllKeys rotates keys for all platforms
func (akm *APIKeyManager) rotateAllKeys() {
	akm.mutex.RLock()
	platforms := make([]string, 0, len(akm.keys))
	for platform := range akm.keys {
		platforms = append(platforms, platform)
	}
	akm.mutex.RUnlock()

	for _, platform := range platforms {
		if err := akm.RotateKeys(platform); err != nil {
			// Log error but continue with other platforms
			fmt.Printf("Failed to rotate keys for platform %s: %v\n", platform, err)
		}
	}
}

// GenerateAPIKey generates a random API key
func GenerateAPIKey(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

// LoadKeysFromConfig loads API keys from configuration
func (akm *APIKeyManager) LoadKeysFromConfig(cfg *Config) error {
	// Load HackerOne keys
	if cfg.APIs.HackerOne.APIKey != "" {
		if err := akm.AddKey("hackerone", cfg.APIs.HackerOne.APIKey, nil); err != nil {
			return fmt.Errorf("failed to add HackerOne key: %w", err)
		}
	}

	// Load BugCrowd keys
	if cfg.APIs.BugCrowd.APIKey != "" {
		if err := akm.AddKey("bugcrowd", cfg.APIs.BugCrowd.APIKey, nil); err != nil {
			return fmt.Errorf("failed to add BugCrowd key: %w", err)
		}
	}

	// Load ChaosDB keys
	if cfg.APIs.ChaosDB.APIKey != "" {
		if err := akm.AddKey("chaosdb", cfg.APIs.ChaosDB.APIKey, nil); err != nil {
			return fmt.Errorf("failed to add ChaosDB key: %w", err)
		}
	}

	return nil
}
