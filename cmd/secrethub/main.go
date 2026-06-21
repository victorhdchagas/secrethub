package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/publiquei/secrethub/internal/server"
)

const version = "0.1.0"

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
  secrethub serve          Start the web server
  secrethub setup          Run the initial setup wizard
  secrethub export <name>  Export a vault as KEY=VALUE
  secrethub list           List available vaults
  secrethub version        Show version
`)
}

func serveCmd() {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 4949, "Server port")
	host := fs.String("host", "127.0.0.1", "Bind address")
	fs.Parse(os.Args[2:])

	if *host == "0.0.0.0" {
		fmt.Fprintln(os.Stderr, "Error: binding to 0.0.0.0 is forbidden for security reasons")
		os.Exit(1)
	}

	cfg := server.Config{
		Host: *host,
		Port: *port,
	}

	if err := server.Serve(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func setupCmd() {
	fmt.Println("Setup wizard not implemented yet")
}

func exportCmd() {
	fmt.Println("Export not implemented yet")
}

func listCmd() {
	fmt.Println("List not implemented yet")
}

func versionCmd() {
	fmt.Printf("secrethub version %s\n", version)
}
