package main

import (
	"fmt"
	"os"
	"syscall"
)

func cmdLock() {
	resp, err := apiRequest("POST", "/vault/lock", nil)
	if err != nil {
		// Try to kill server directly
		if pid, pidErr := readPID(); pidErr == nil {
			if p, findErr := os.FindProcess(pid); findErr == nil {
				p.Signal(syscall.SIGTERM)
			}
		}
		removeSessionToken()
		removePID()
		fmt.Println("Vault locked (server stopped).")
		return
	}
	resp.Body.Close()

	// Kill the server process
	if pid, err := readPID(); err == nil {
		if p, err := os.FindProcess(pid); err == nil {
			p.Signal(syscall.SIGTERM)
		}
	}

	removeSessionToken()
	removePID()
	fmt.Println("Vault locked.")
}
