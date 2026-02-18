package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "unlock":
		cmdUnlock()
	case "lock":
		cmdLock()
	case "serve":
		cmdServe()
	case "status":
		cmdStatus()
	case "schema":
		cmdSchema()
	case "set":
		cmdSet()
	case "get":
		cmdGet()
	case "list":
		cmdList()
	case "delete":
		cmdDelete()
	case "set-sensitivity":
		cmdSetSensitivity()
	case "export":
		cmdExport()
	case "audit":
		cmdAudit()
	case "create-service-token":
		cmdCreateServiceToken()
	case "list-service-tokens":
		cmdListServiceTokens()
	case "revoke-service-token":
		cmdRevokeServiceToken()
	case "onboard":
		cmdOnboard()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: pvault <command> [args]

Commands:
  onboard               Create vault, unlock, and populate common fields
  init                  Create a new vault
  unlock                Unlock vault (starts background server)
  lock                  Lock vault (stops server)
  serve                 Run server in foreground
  status                Show vault status
  schema                Show recommended field names (--json for raw JSON)
  set <id> <value>      Set a field (e.g., identity.full_name "Cool Cucumber")
  get <id>              Get a field value
  list [category]       List fields
  delete <id>           Delete a field
  set-sensitivity <id> <tier>  Set field sensitivity (public|standard|sensitive|critical)
  export                Export all decrypted fields as JSON
  audit                 Show access audit log
  create-service-token <consumer>  Create a long-lived service token
  list-service-tokens              List active service tokens
  revoke-service-token <prefix>    Revoke a service token by prefix`)
}
