package main

import (
	"fmt"
	"os"
	"strings"
)

func cmdSet() {
	if len(os.Args) < 4 {
		fatal("usage: pvault set <id> <value>\n  example: pvault set identity.full_name \"Cool Cucumber\"")
	}
	id := os.Args[2]
	value := strings.Join(os.Args[3:], " ")

	if !strings.Contains(id, ".") {
		fatal("field ID must be category.name (e.g., identity.full_name)")
	}

	resp, err := apiRequest("PUT", "/vault/fields/"+id, map[string]string{
		"value": value,
	})
	if err != nil {
		fatal("request failed: %v", err)
	}

	var result struct {
		Status     string `json:"status"`
		Suggestion *struct {
			Canonical   string `json:"canonical"`
			Description string `json:"description"`
		} `json:"suggestion,omitempty"`
	}
	if err := apiResult(resp, &result); err != nil {
		fatal("%v", err)
	}
	fmt.Printf("Set %s\n", id)
	if result.Suggestion != nil {
		fmt.Fprintf(os.Stderr, "Hint: the recommended field is %s (%s)\n",
			result.Suggestion.Canonical, result.Suggestion.Description)
	}
}
