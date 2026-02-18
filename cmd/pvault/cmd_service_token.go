package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func cmdCreateServiceToken() {
	if len(os.Args) < 3 {
		fatal("usage: pvault create-service-token <consumer> [--scope categories] [--ttl duration]")
	}

	consumer := os.Args[2]
	scope := "*"
	ttl := "8760h" // 1 year

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--scope":
			if i+1 < len(os.Args) {
				scope = os.Args[i+1]
				i++
			}
		case "--ttl":
			if i+1 < len(os.Args) {
				ttl = os.Args[i+1]
				i++
			}
		}
	}

	resp, err := apiRequest("POST", "/vault/tokens/service", map[string]string{
		"consumer": consumer,
		"scope":    scope,
		"ttl":      ttl,
	})
	if err != nil {
		fatal("request failed: %v", err)
	}

	var result struct {
		Token     string `json:"token"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := apiResult(resp, &result); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Service token created for %q\n", consumer)
	fmt.Printf("Token:   %s\n", result.Token)
	fmt.Printf("Scope:   %s\n", scope)
	fmt.Printf("Expires: %s\n", result.ExpiresAt)
	fmt.Println("\nSave this token — it cannot be displayed again.")
}

func cmdListServiceTokens() {
	resp, err := apiRequest("GET", "/vault/tokens/service", nil)
	if err != nil {
		fatal("request failed: %v", err)
	}

	var tokens []struct {
		TokenPrefix string `json:"token_prefix"`
		Consumer    string `json:"consumer"`
		Scope       string `json:"scope"`
		ExpiresAt   string `json:"expires_at"`
		CreatedAt   string `json:"created_at"`
	}
	if err := apiResult(resp, &tokens); err != nil {
		fatal("%v", err)
	}

	if len(tokens) == 0 {
		fmt.Println("No service tokens.")
		return
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(tokens)
}

func cmdRevokeServiceToken() {
	if len(os.Args) < 3 {
		fatal("usage: pvault revoke-service-token <prefix>")
	}
	token := os.Args[2]

	resp, err := apiRequest("DELETE", "/vault/tokens/service/"+token, nil)
	if err != nil {
		fatal("request failed: %v", err)
	}

	var result struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	if err := apiResult(resp, &result); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Revoked %d token(s).\n", result.Count)
}
