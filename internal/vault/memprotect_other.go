//go:build !linux && !darwin

package vault

func lockMemory(b []byte)   {}
func unlockMemory(b []byte) {}
func disableCoreDumps()     {}
