package main

import (
	"fmt"
	"os"
)

func cmdSetSensitivity() {
	if len(os.Args) < 4 {
		fatal("usage: pvault set-sensitivity <id> <tier>\n  tiers: public, standard, sensitive, critical")
	}
	id := os.Args[2]
	tier := os.Args[3]

	valid := map[string]bool{"public": true, "standard": true, "sensitive": true, "critical": true}
	if !valid[tier] {
		fatal("invalid tier %q (must be public, standard, sensitive, or critical)", tier)
	}

	resp, err := apiRequest("PUT", "/vault/sensitivity/"+id, map[string]string{
		"tier": tier,
	})
	if err != nil {
		fatal("request failed: %v", err)
	}

	var result map[string]string
	if err := apiResult(resp, &result); err != nil {
		fatal("%v", err)
	}
	fmt.Printf("Set %s sensitivity to %s\n", id, tier)
}
