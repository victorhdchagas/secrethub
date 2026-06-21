package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/publiquei/secrethub/internal/vault"
)

func exportCmd() {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	dotenv := fs.Bool("dotenv", false, "Write to .env file instead of stdout")
	run := fs.Bool("run", false, "Execute a command with vault vars as env")
	fs.Parse(os.Args[2:])

	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: secrethub export <vault-name> [--dotenv] [--run <cmd...>]")
		os.Exit(1)
	}
	name := args[0]

	dir := secrethubDir()

	salt, err := os.ReadFile(filepath.Join(dir, "salt"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Setup incomplete: salt file missing. Run 'secrethub setup --force'")
		os.Exit(1)
	}

	password := promptPassword("Master password")
	vk := vault.DeriveKey(password, salt)

	store := vault.NewStore(filepath.Join(dir, "vaults"))
	data, err := store.Load(context.Background(), name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Vault not found:", name)
		os.Exit(1)
	}

	plaintext, err := vault.Decrypt(data, vk)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Decryption failed — wrong password?")
		os.Exit(1)
	}

	v, err := vault.DeserializeVault(name, plaintext)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Corrupt vault")
		os.Exit(1)
	}

	if *run {
		runExport(v, args[1:])
		return
	}

	output := v.Export()

	if *dotenv {
		envPath := name + ".env"
		if err := os.WriteFile(envPath, []byte(output), 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing .env: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Written to", envPath)
		return
	}

	fmt.Print(output)
}

func runExport(v *vault.Vault, args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: secrethub export <name> --run <command...>")
		os.Exit(1)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()
	for k, val := range v.All() {
		cmd.Env = append(cmd.Env, k+"="+val)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
