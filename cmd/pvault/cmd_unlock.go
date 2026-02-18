package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func cmdUnlock() {
	// Probe the port first — catches stale servers even if the PID file is gone.
	if portHasVault() {
		if isVaultUnlocked() {
			fmt.Println("Vault is already unlocked (server running).")
			return
		}
		// Server running but vault auto-locked — re-unlock via API
		pw, err := promptPassword("Profile password: ")
		if err != nil {
			fatal("reading password: %v", err)
		}
		sk, err := readSecretKey()
		if err != nil {
			fatal("%v", err)
		}
		reUnlock(pw, sk)
		return
	}

	// Nothing on the port — clean up any stale PID file
	removePID()

	pw, err := promptPassword("Profile password: ")
	if err != nil {
		fatal("reading password: %v", err)
	}

	sk, err := readSecretKey()
	if err != nil {
		fatal("%v", err)
	}

	// Start background server
	exe, err := os.Executable()
	if err != nil {
		fatal("finding executable: %v", err)
	}

	cmd := exec.Command(exe, "serve", "--password-stdin")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("VAULT_DIR=%s", vaultDir()),
	)

	// Pass credentials via stdin pipe
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fatal("creating stdin pipe: %v", err)
	}

	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		fatal("starting server: %v", err)
	}

	// Write credentials and close pipe
	fmt.Fprintf(stdin, "%s\n%s\n", pw, sk)
	stdin.Close()

	writePID(cmd.Process.Pid)

	// Wait briefly for server to be ready
	time.Sleep(500 * time.Millisecond)

	// Verify it's running AND unlocked
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Vault server started (could not verify status).")
			return
		default:
			resp, err := apiRequest("GET", "/vault/status", nil)
			if err == nil {
				var status struct {
					Locked bool `json:"locked"`
				}
				json.NewDecoder(resp.Body).Decode(&status)
				resp.Body.Close()
				if !status.Locked {
					fmt.Println("Vault unlocked. Server running on", serverAddr())
					return
				}
				// Server responded but vault is locked — our spawn likely
				// failed and a stale server answered. Kill it and retry.
				fatal("server on %s is not the one we started (vault still locked) — run 'pvault lock' then 'pvault unlock'", serverAddr())
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// portHasVault probes the server address with GET /vault/status.
// Returns true if a vault server responds, false otherwise.
func portHasVault() bool {
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(serverAddr() + "/vault/status")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func isVaultUnlocked() bool {
	resp, err := apiRequest("GET", "/vault/status", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var status struct {
		Locked bool `json:"locked"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return false
	}
	return !status.Locked
}

func reUnlock(password, secretKey string) {
	body := map[string]string{
		"password":   password,
		"secret_key": secretKey,
	}
	resp, err := apiRequest("POST", "/vault/unlock", body)
	if err != nil {
		fatal("re-unlock request: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Token string `json:"token"`
		Error string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if resp.StatusCode >= 400 {
		fatal("unlock failed: %s", result.Error)
	}
	if result.Token == "" {
		fatal("unlock returned no token")
	}

	if err := writeSessionToken(result.Token); err != nil {
		fatal("write session: %v", err)
	}
	fmt.Println("Vault unlocked. Server running on", serverAddr())
}
