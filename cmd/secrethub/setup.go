package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/victorhdchagas/secrethub/internal/auth"
	"github.com/victorhdchagas/secrethub/internal/vault"
)

func setupCmd() {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	force := fs.Bool("force", false, "Overwrite existing setup")
	_ = fs.Parse(os.Args[2:]) // intentionally discarded — flag.ExitOnError

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
