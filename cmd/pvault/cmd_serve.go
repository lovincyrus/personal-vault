package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lovincyrus/personal-vault/internal/api"
	"github.com/lovincyrus/personal-vault/internal/vault"
)

var passwordFromStdin bool

func init() {
	for _, arg := range os.Args {
		if arg == "--password-stdin" {
			passwordFromStdin = true
		}
	}
}

func cmdServe() {
	dir := vaultDir()
	v, err := vault.Open(dir)
	if err != nil {
		fatal("open vault: %v", err)
	}
	defer v.Close()

	// Get credentials from stdin pipe (sent by unlock command) or prompt
	var pw, sk string
	if passwordFromStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			pw = strings.TrimSpace(scanner.Text())
		}
		if scanner.Scan() {
			sk = strings.TrimSpace(scanner.Text())
		}
		if pw == "" || sk == "" {
			fatal("failed to read credentials from stdin")
		}
	} else {
		pw, err = promptPassword("Profile password: ")
		if err != nil {
			fatal("reading password: %v", err)
		}
		sk, err = readSecretKey()
		if err != nil {
			fatal("%v", err)
		}
	}

	token, err := v.Unlock(pw, sk)
	if err != nil {
		fatal("unlock: %v", err)
	}

	// Note: Go strings are immutable; setting pw="" does not zero heap memory.
	// Accept this limitation — use []byte for passwords if zeroing is critical.

	addr := "127.0.0.1:7200"
	if a := os.Getenv("VAULT_PORT"); a != "" {
		addr = "127.0.0.1:" + a
	}

	srv := api.New(v, addr)
	ln, err := srv.Start()
	if err != nil {
		fatal("start server: %v", err)
	}

	// Write session token only after server binds successfully.
	// Writing before srv.Start() causes token mismatch if the port is occupied
	// by a stale process — the file gets a new token while the old server
	// still holds the previous one.
	if err := writeSessionToken(token); err != nil {
		fatal("write session: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Vault server listening on %s\n", ln.Addr())

	// Wait for signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Fprintln(os.Stderr, "\nShutting down...")
	v.Lock()
	srv.Stop(context.Background())
	removeSessionToken()
	removePID()
}
