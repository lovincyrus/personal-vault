//go:build linux || darwin

package vault

import "syscall"

// lockMemory locks the byte slice's memory page(s) to prevent swapping to disk.
// Best-effort: failure is silently ignored (process may lack CAP_IPC_LOCK).
func lockMemory(b []byte) {
	_ = syscall.Mlock(b)
}

// unlockMemory unlocks previously locked memory pages.
// Best-effort: failure is silently ignored.
func unlockMemory(b []byte) {
	_ = syscall.Munlock(b)
}

// disableCoreDumps sets RLIMIT_CORE to 0 to prevent key material from appearing in core dumps.
// Best-effort: failure is silently ignored.
func disableCoreDumps() {
	_ = syscall.Setrlimit(syscall.RLIMIT_CORE, &syscall.Rlimit{Cur: 0, Max: 0})
}
