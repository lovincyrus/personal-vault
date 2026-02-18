package main

import (
	"fmt"

	"github.com/lovincyrus/personal-vault/internal/vault"
)

func cmdStatus() {
	resp, err := apiRequest("GET", "/vault/status", nil)
	if err != nil {
		fmt.Println("Vault is locked (server not running).")
		return
	}

	var status vault.VaultStatus
	if err := apiResult(resp, &status); err != nil {
		fatal("%v", err)
	}

	if !status.Initialized {
		fmt.Println("Vault is not initialized. Run 'pvault init' first.")
		return
	}

	if status.Locked {
		fmt.Println("Status:  locked")
	} else {
		fmt.Println("Status:  unlocked")
	}
	fmt.Printf("Fields:  %d\n", status.FieldCount)
	if len(status.Categories) > 0 {
		fmt.Println("Categories:")
		for cat, count := range status.Categories {
			fmt.Printf("  %-20s %d fields\n", cat, count)
		}
	}
}
