package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/publiquei/secrethub/internal/vault"
)

func listCmd() {
	dir := secrethubDir()
	store := vault.NewStore(filepath.Join(dir, "vaults"))
	names, err := store.List(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error listing vaults:", err)
		os.Exit(1)
	}

	if len(names) == 0 {
		fmt.Println("No vaults found.")
		return
	}

	for _, name := range names {
		fmt.Println(name)
	}
}
