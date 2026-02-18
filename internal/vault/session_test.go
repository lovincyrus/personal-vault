package vault

import (
	"testing"
)

func TestMemProtect_NoPanic(t *testing.T) {
	// Verify lockMemory/unlockMemory/disableCoreDumps don't panic.
	// These are best-effort and may silently fail without CAP_IPC_LOCK,
	// but they must never crash the process.
	b := make([]byte, 32)
	lockMemory(b)
	unlockMemory(b)
	disableCoreDumps()
}

func TestSession_MemProtect_Integration(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")

	s, err := NewSession(key, func() {})
	if err != nil {
		t.Fatal(err)
	}

	// Key should be accessible
	got := s.VaultKey()
	if got == nil {
		t.Fatal("expected non-nil vault key")
	}
	if string(got) != string(key) {
		t.Fatal("vault key mismatch")
	}

	// Destroy should zero and unlock without panic
	s.Destroy()

	if s.VaultKey() != nil {
		t.Fatal("expected nil vault key after destroy")
	}
	if s.ValidateToken(s.Token()) {
		t.Fatal("expected invalid token after destroy")
	}
}

func TestSession_AutoLock_MemProtect(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	locked := make(chan struct{})

	s, err := NewSession(key, func() { close(locked) })
	if err != nil {
		t.Fatal(err)
	}

	// Shorten TTL to trigger auto-lock quickly
	s.mu.Lock()
	s.ttl = 1
	s.timer.Reset(1)
	s.mu.Unlock()

	<-locked

	// After auto-lock, key should be zeroed (unlockMemory called in zeroKey)
	if s.VaultKey() != nil {
		t.Fatal("expected nil vault key after auto-lock")
	}
}
