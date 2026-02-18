package main

import (
	"fmt"

	"github.com/lovincyrus/personal-vault/internal/store"
)

func cmdAudit() {
	resp, err := apiRequest("GET", "/vault/audit?limit=20", nil)
	if err != nil {
		fatal("request failed: %v", err)
	}

	var entries []store.AuditEntry
	if err := apiResult(resp, &entries); err != nil {
		fatal("%v", err)
	}

	if len(entries) == 0 {
		fmt.Println("No audit entries.")
		return
	}

	for _, e := range entries {
		purpose := ""
		if e.Purpose != "" {
			purpose = fmt.Sprintf(" (%s)", e.Purpose)
		}
		fmt.Printf("%-20s %-10s %-8s %s%s\n",
			e.CreatedAt.Format("2006-01-02 15:04:05"),
			e.Consumer, e.Action, e.Scope, purpose)
	}
}
