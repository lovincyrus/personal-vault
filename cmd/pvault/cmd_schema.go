package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lovincyrus/personal-vault/internal/vault"
)

func cmdSchema() {
	jsonFlag := false
	for _, arg := range os.Args[2:] {
		if arg == "--json" {
			jsonFlag = true
		}
	}

	if jsonFlag {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(vault.RecommendedSchema)
		return
	}

	fmt.Println("Recommended Vault Schema")
	fmt.Println("========================")
	fmt.Println()
	for _, cat := range vault.RecommendedSchema.Categories {
		fmt.Printf("%s — %s\n", cat.Name, cat.Description)
		if len(cat.Fields) == 0 {
			fmt.Println("  (user-defined fields)")
		}
		for _, f := range cat.Fields {
			fmt.Printf("  %-35s %s", f.ID, f.Description)
			if f.Sensitivity != "standard" {
				fmt.Printf(" [%s]", f.Sensitivity)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}
