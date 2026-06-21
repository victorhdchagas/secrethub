package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/publiquei/secrethub/internal/auth"
	"github.com/publiquei/secrethub/internal/vault"
)

func tokenCmd() {
	if len(os.Args) < 3 {
		fmt.Fprint(os.Stderr, `Usage:
  secrethub token create          Create a new machine token
  secrethub token revoke <prefix> Revoke a token by its prefix
  secrethub token list            List all tokens
`)
		os.Exit(1)
	}

	sub := os.Args[2]
	switch sub {
	case "create":
		tokenCreateCmd()
	case "revoke":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: secrethub token revoke <prefix>")
			os.Exit(1)
		}
		tokenRevokeCmd(os.Args[3])
	case "list":
		tokenListCmd()
	default:
		fmt.Fprintf(os.Stderr, "Unknown token subcommand: %s\n", sub)
		os.Exit(1)
	}
}

func tokenCreateCmd() {
	dir := secrethubDir()
	ctx := context.Background()

	password := promptPassword("Master password")

	salt, err := os.ReadFile(filepath.Join(dir, "salt"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erro: setup incompleto — refaça 'secrethub setup'")
		os.Exit(1)
	}

	vk := vault.DeriveKey(password, salt)

	th := auth.NewTokenHandler(filepath.Join(dir, "machine.tokens"))
	if err := th.Load(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Erro ao carregar tokens:", err)
		os.Exit(1)
	}

	token, err := th.Generate(ctx, vk)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erro ao gerar token:", err)
		os.Exit(1)
	}

	fmt.Println("Token gerado com sucesso!")
	fmt.Println()
	fmt.Println("Token (mostrado uma única vez):")
	fmt.Println(token)
	fmt.Println()
	fmt.Println("Prefixo:", token[:8])
	fmt.Println("Use em CI/CD via:")
	fmt.Println("  curl http://<server>:<port>/api/vault/<name>/export?token=" + token)
}

func tokenRevokeCmd(prefix string) {
	dir := secrethubDir()
	ctx := context.Background()

	th := auth.NewTokenHandler(filepath.Join(dir, "machine.tokens"))
	if err := th.Load(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Erro ao carregar tokens:", err)
		os.Exit(1)
	}

	if err := th.Revoke(ctx, prefix); err != nil {
		fmt.Fprintln(os.Stderr, "Erro:", err)
		os.Exit(1)
	}

	fmt.Println("Token", prefix, "revogado.")
}

func tokenListCmd() {
	dir := secrethubDir()
	ctx := context.Background()

	th := auth.NewTokenHandler(filepath.Join(dir, "machine.tokens"))
	if err := th.Load(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "Erro ao carregar tokens:", err)
		os.Exit(1)
	}

	infos, err := th.List(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Erro:", err)
		os.Exit(1)
	}

	if len(infos) == 0 {
		fmt.Println("Nenhum token cadastrado.")
		return
	}

	fmt.Println("Tokens ativos:")
	for _, info := range infos {
		fmt.Printf("  %s  (criado em %s)\n", info.Prefix, info.CreatedAt.Format(time.RFC3339))
	}
}
