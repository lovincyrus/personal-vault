package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lovincyrus/personal-vault/internal/vault"
)

func cmdExport() {
	resp, err := apiRequest("GET", "/vault/context", nil)
	if err != nil {
		fatal("request failed: %v", err)
	}

	var ctx vault.ContextBundle
	if err := apiResult(resp, &ctx); err != nil {
		fatal("%v", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding: %v\n", err)
	}
}
