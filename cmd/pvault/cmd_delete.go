package main

import (
	"fmt"
	"os"
)

func cmdDelete() {
	if len(os.Args) < 3 {
		fatal("usage: pvault delete <id>")
	}
	id := os.Args[2]

	resp, err := apiRequest("DELETE", "/vault/fields/"+id, nil)
	if err != nil {
		fatal("request failed: %v", err)
	}

	var result map[string]string
	if err := apiResult(resp, &result); err != nil {
		fatal("%v", err)
	}
	fmt.Printf("Deleted %s\n", id)
}
