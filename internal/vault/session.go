package vault

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"sync"
	"time"
)

const defaultAutoLockDuration = 30 * time.Minute

// Session holds the in-memory vault key and session token.
type Session struct {
	mu       sync.Mutex
	token    string
	vaultKey []byte
	timer    *time.Timer
	lockFn   func()
	ttl      time.Duration
}

// NewSession creates a session with the given vault key and auto-lock callback.
func NewSession(vaultKey []byte, lockFn func()) (*Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}

	s := &Session{
		token:  hex.EncodeToString(tokenBytes),
		lockFn: lockFn,
		ttl:    defaultAutoLockDuration,
	}
	// Copy vault key so caller can't mutate it
	s.vaultKey = make([]byte, len(vaultKey))
	copy(s.vaultKey, vaultKey)
	lockMemory(s.vaultKey)
	disableCoreDumps()

	s.timer = time.AfterFunc(s.ttl, s.autoLock)
	return s, nil
}

// Token returns the session token string.
func (s *Session) Token() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

// VaultKey returns a copy of the vault key. Returns nil if session is destroyed.
func (s *Session) VaultKey() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.vaultKey == nil {
		return nil
	}
	cp := make([]byte, len(s.vaultKey))
	copy(cp, s.vaultKey)
	return cp
}

// ValidateToken checks a token using constant-time comparison.
func (s *Session) ValidateToken(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(s.token), []byte(token)) == 1
}

// Touch resets the auto-lock timer.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.timer != nil {
		s.timer.Reset(s.ttl)
	}
}

// Destroy zeroes the vault key and invalidates the session.
func (s *Session) Destroy() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.zeroKey()
	s.token = ""
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
}

func (s *Session) autoLock() {
	s.mu.Lock()
	s.zeroKey()
	s.token = ""
	s.timer = nil
	lockFn := s.lockFn
	s.mu.Unlock()

	if lockFn != nil {
		lockFn()
	}
}

func (s *Session) zeroKey() {
	unlockMemory(s.vaultKey)
	for i := range s.vaultKey {
		s.vaultKey[i] = 0
	}
	s.vaultKey = nil
}
