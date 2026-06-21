package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

var version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	sub := os.Args[1]
	switch sub {
	case "serve":
		serveCmd()
	case "setup":
		setupCmd()
	case "export":
		exportCmd()
	case "list":
		listCmd()
	case "token":
		tokenCmd()
	case "version":
		versionCmd()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", sub)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprint(os.Stderr, `SecretHub - Gerenciador de Secrets Pessoal

Usage:
  secrethub serve              Start the web server
  secrethub setup              Run the initial setup wizard
  secrethub export <name>      Export a vault as KEY=VALUE
  secrethub list               List available vaults
  secrethub token <subcommand> Manage machine tokens (create/revoke/list)
  secrethub version            Show version
`)
}

func secrethubDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".secrethub")
}

var stdinScanner = bufio.NewScanner(os.Stdin)

func promptPassword(label string) string {
	fmt.Fprint(os.Stderr, label+": ")
	stdinScanner.Scan()
	return stdinScanner.Text()
}

func versionCmd() {
	fmt.Printf("secrethub version %s\n", version)
}
