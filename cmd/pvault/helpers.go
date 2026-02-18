package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func vaultDir() string {
	if d := os.Getenv("VAULT_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pvault")
}

func serverAddr() string {
	if a := os.Getenv("VAULT_ADDR"); a != "" {
		return a
	}
	return "http://127.0.0.1:7200"
}

func sessionPath() string {
	return filepath.Join(vaultDir(), ".session")
}

func pidPath() string {
	return filepath.Join(vaultDir(), "pvault.pid")
}

func secretKeyPath() string {
	return filepath.Join(vaultDir(), "secret.key")
}

func readSessionToken() (string, error) {
	data, err := os.ReadFile(sessionPath())
	if err != nil {
		return "", fmt.Errorf("vault is not unlocked (no session file)")
	}
	return strings.TrimSpace(string(data)), nil
}

func writeSessionToken(token string) error {
	return os.WriteFile(sessionPath(), []byte(token+"\n"), 0600)
}

func removeSessionToken() {
	os.Remove(sessionPath())
}

func readSecretKey() (string, error) {
	data, err := os.ReadFile(secretKeyPath())
	if err != nil {
		return "", fmt.Errorf("secret key not found at %s", secretKeyPath())
	}
	return strings.TrimSpace(string(data)), nil
}

func writePID(pid int) error {
	return os.WriteFile(pidPath(), []byte(strconv.Itoa(pid)+"\n"), 0600)
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidPath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func removePID() {
	os.Remove(pidPath())
}

func promptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(pw), nil
}

// apiRequest makes an authenticated HTTP request to the vault server.
func apiRequest(method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(buf)
	}

	url := serverAddr() + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	token, err := readSessionToken()
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return http.DefaultClient.Do(req)
}

// apiResult decodes a JSON response or returns the error.
func apiResult(resp *http.Response, target any) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return fmt.Errorf("%s", errResp.Error)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if target != nil {
		return json.NewDecoder(resp.Body).Decode(target)
	}
	return nil
}

func fatal(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
