package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

func cmdUI() {
	token, err := readSessionToken()
	if err != nil {
		fatal("vault is not unlocked — run 'pvault unlock' first")
	}

	// Verify vault is reachable and unlocked
	resp, err := apiRequest("GET", "/vault/status", nil)
	if err != nil {
		fatal("cannot reach vault server — is it running? Try 'pvault unlock'")
	}
	resp.Body.Close()

	url := serverAddr() + "/ui#token=" + token

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}

	if cmd != nil {
		if err := cmd.Start(); err == nil {
			fmt.Println("Opened vault UI in your browser.")
			return
		}
	}

	// Fallback: print URL
	fmt.Println("Open this URL in your browser:")
	fmt.Println()
	fmt.Println("  " + url)
	fmt.Println()
}
