package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/publiquei/secrethub/internal/auth"
	"github.com/publiquei/secrethub/internal/server"
	"github.com/publiquei/secrethub/internal/vault"
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
		Host:    *host,
		Port:    *port,
		DataDir: secrethubDir(),
	}

	if err := server.Serve(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func setupCmd() {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	force := fs.Bool("force", false, "Overwrite existing setup")
	fs.Parse(os.Args[2:])

	dir := secrethubDir()
	ctx := context.Background()

	if _, err := os.Stat(dir); err == nil && !*force {
		fmt.Fprint(os.Stderr, "SecretHub is already set up.\nDelete ~/.secrethub/ to reset, or use --force to overwrite.\n")
		os.Exit(1)
	}

	fmt.Println("=== SecretHub Setup ===")

	password := promptPassword("Master password")
	confirm := promptPassword("Confirm master password")
	if password != confirm {
		fmt.Fprintln(os.Stderr, "Passwords do not match")
		os.Exit(1)
	}

	hasher := auth.NewBCryptHasher(12)
	hash, err := hasher.Hash(ctx, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error hashing password: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nConfiguring Two-Factor Authentication...")
	totp := auth.NewTOTPHandler()
	key, err := totp.Generate(ctx, "secrethub")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating TOTP: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nScan the QR code below with Google Authenticator or Authy:")
	fmt.Println(key.URL)
	fmt.Println("\nOr enter this key manually:")
	fmt.Println(key.Secret)

	fmt.Println("\nGenerating recovery codes...")
	recovery := auth.NewRecoveryHandler(nil)
	codes, err := recovery.Generate(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating recovery codes: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nStore these recovery codes in a safe place (paper, password manager):")
	for i, code := range codes {
		fmt.Printf("  %2d. %s\n", i+1, code)
	}
	fmt.Println("\nEach recovery code can only be used once.")

	if err := os.MkdirAll(filepath.Join(dir, "vaults"), 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filepath.Join(dir, "master.hash"), []byte(hash), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving master.hash: %v\n", err)
		os.Exit(1)
	}

	// TODO: Encrypt totp.secret with master password (Prompt 3)
	if err := os.WriteFile(filepath.Join(dir, "totp.secret"), []byte(key.Secret), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving totp.secret: %v\n", err)
		os.Exit(1)
	}

	hashes := recovery.Hashes()
	var hashData []byte
	for _, h := range hashes {
		hashData = append(hashData, []byte(h+"\n")...)
	}
	if err := os.WriteFile(filepath.Join(dir, "recovery.hashes"), hashData, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving recovery.hashes: %v\n", err)
		os.Exit(1)
	}

	salt, err := vault.NewSalt()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating salt: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(dir, "salt"), salt, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving salt: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Setup complete! Run 'secrethub serve' to start the dashboard.")
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

func exportCmd() {
	fmt.Println("Export not implemented yet")
}

func listCmd() {
	fmt.Println("List not implemented yet")
}

func versionCmd() {
	fmt.Printf("secrethub version %s\n", version)
}
