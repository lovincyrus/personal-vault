package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/lovincyrus/personal-vault/internal/vault"
)

var onboardFields = []struct {
	prompt  string
	fieldID string
}{
	{"Full name", "identity.full_name"},
	{"Email", "identity.email"},
	{"City", "addresses.home_city"},
	{"State", "addresses.home_state"},
	{"ZIP", "addresses.home_zip"},
	{"Country", "addresses.home_country"},
	{"Timezone", "preferences.timezone"},
}

func cmdOnboard() {
	dir := vaultDir()

	// Check if vault already exists
	if _, err := os.Stat(dir + "/vault.db"); err == nil {
		fatal("vault already initialized at %s — use 'pvault unlock' instead", dir)
	}

	fmt.Println("Create your vault")

	pw, err := promptPassword("  Profile password: ")
	if err != nil {
		fatal("reading password: %v", err)
	}
	if len(pw) < 8 {
		fatal("password must be at least 8 characters")
	}

	confirm, err := promptPassword("  Confirm: ")
	if err != nil {
		fatal("reading confirmation: %v", err)
	}
	if pw != confirm {
		fatal("passwords do not match")
	}

	sk, err := vault.Init(dir, pw)
	if err != nil {
		fatal("%v", err)
	}

	fmt.Println()
	fmt.Println("Your secret key (save this somewhere safe):")
	fmt.Printf("  %s\n", sk)
	fmt.Println()

	// Start background server (same pattern as cmdUnlock)
	exe, err := os.Executable()
	if err != nil {
		fatal("finding executable: %v", err)
	}

	cmd := exec.Command(exe, "serve", "--password-stdin")
	cmd.Env = append(os.Environ(), fmt.Sprintf("VAULT_DIR=%s", dir))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fatal("creating stdin pipe: %v", err)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		fatal("starting server: %v", err)
	}

	fmt.Fprintf(stdin, "%s\n%s\n", pw, sk)
	stdin.Close()

	writePID(cmd.Process.Pid)

	// Wait for server to be ready
	time.Sleep(500 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ready := false
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
					ready = true
				}
			}
		}
		if ready {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("Vault unlocked. Server running on %s\n", serverAddr())
	fmt.Println()

	// Prompt for common fields
	fmt.Println("Let's add some basics (press Enter to skip any):")
	reader := bufio.NewReader(os.Stdin)
	saved := 0

	for _, f := range onboardFields {
		fmt.Printf("  %s: ", f.prompt)
		line, _ := reader.ReadString('\n')
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}

		resp, err := apiRequest("PUT", "/vault/fields/"+f.fieldID, map[string]string{
			"value": value,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: could not save %s: %v\n", f.fieldID, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 400 {
			fmt.Fprintf(os.Stderr, "  Warning: could not save %s (HTTP %d)\n", f.fieldID, resp.StatusCode)
			continue
		}
		saved++
	}

	fmt.Println()
	if saved > 0 {
		fmt.Printf("Done — %d field(s) saved. Your vault is ready.\n", saved)
	} else {
		fmt.Println("Done — your vault is ready.")
	}
	fmt.Println("Run 'pvault status' to see what's stored.")
}
