package main

import (
	"fmt"
	"os"

	"github.com/lovincyrus/personal-vault/internal/vault"
)

func cmdGet() {
	if len(os.Args) < 3 {
		fatal("usage: pvault get <id>\n  example: pvault get identity.full_name")
	}
	id := os.Args[2]

	resp, err := apiRequest("GET", "/vault/fields/"+id, nil)
	if err != nil {
		fatal("request failed: %v", err)
	}

	var field vault.FieldInfo
	if err := apiResult(resp, &field); err != nil {
		fatal("%v", err)
	}
	fmt.Println(field.Value)
}
