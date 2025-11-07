package evm

import (
	"sync"
	"time"
)

// NonceEntry tracks when a nonce was first seen and its expiration
type NonceEntry struct {
	FirstSeen time.Time
	ExpiresAt time.Time
}

// NonceStore tracks used ERC-3009 nonces to prevent replay attacks
// This is an optimization layer - the smart contract also enforces nonce uniqueness
type NonceStore struct {
	mu     sync.RWMutex
	nonces map[string]NonceEntry // key: "from_address:nonce_hex"

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewNonceStore creates a new nonce tracking store
func NewNonceStore() *NonceStore {
	ns := &NonceStore{
		nonces:      make(map[string]NonceEntry),
		stopCleanup: make(chan bool),
	}

	// Start cleanup goroutine (runs every 5 minutes)
	ns.cleanupTicker = time.NewTicker(5 * time.Minute)
	go ns.cleanupExpiredNonces()

	return ns
}

// IsNonceUsed checks if a nonce has already been seen for a given address
func (ns *NonceStore) IsNonceUsed(fromAddress, nonce string) bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	key := fromAddress + ":" + nonce
	entry, exists := ns.nonces[key]

	if !exists {
		return false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return false
	}

	return true
}

// MarkNonceUsed records that a nonce has been used
// validBefore is the Unix timestamp when the authorization expires
func (ns *NonceStore) MarkNonceUsed(fromAddress, nonce string, validBefore int64) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	key := fromAddress + ":" + nonce

	// Store nonce with expiration = validBefore + 1 hour buffer
	expiresAt := time.Unix(validBefore, 0).Add(1 * time.Hour)

	ns.nonces[key] = NonceEntry{
		FirstSeen: time.Now(),
		ExpiresAt: expiresAt,
	}
}

// cleanupExpiredNonces removes expired nonces from the store periodically
func (ns *NonceStore) cleanupExpiredNonces() {
	for {
		select {
		case <-ns.cleanupTicker.C:
			ns.mu.Lock()
			now := time.Now()
			for key, entry := range ns.nonces {
				if now.After(entry.ExpiresAt) {
					delete(ns.nonces, key)
				}
			}
			ns.mu.Unlock()
		case <-ns.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup goroutine
func (ns *NonceStore) Stop() {
	ns.cleanupTicker.Stop()
	ns.stopCleanup <- true
}

// GetStats returns statistics about the nonce store (for monitoring/debugging)
func (ns *NonceStore) GetStats() map[string]interface{} {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	total := len(ns.nonces)
	expired := 0
	now := time.Now()

	for _, entry := range ns.nonces {
		if now.After(entry.ExpiresAt) {
			expired++
		}
	}

	return map[string]interface{}{
		"total_nonces":   total,
		"active_nonces":  total - expired,
		"expired_nonces": expired,
	}
}
