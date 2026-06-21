package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/victorhdchagas/secrethub/internal/server"
)

func serveCmd() {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 4949, "Server port")
	host := fs.String("host", "127.0.0.1", "Bind address")
	tlsCert := fs.String("tls-cert", "", "TLS certificate file (enables HTTPS)")
	tlsKey := fs.String("tls-key", "", "TLS private key file (enables HTTPS)")
	_ = fs.Parse(os.Args[2:]) // intentionally discarded — flag.ExitOnError

	if *host == "0.0.0.0" {
		fmt.Fprintln(os.Stderr, "Error: binding to 0.0.0.0 is forbidden for security reasons")
		os.Exit(1)
	}

	if (*tlsCert == "") != (*tlsKey == "") {
		fmt.Fprintln(os.Stderr, "Error: --tls-cert and --tls-key must be used together")
		os.Exit(1)
	}

	cfg := server.Config{
		Host:        *host,
		Port:        *port,
		DataDir:     secrethubDir(),
		TLSCertFile: *tlsCert,
		TLSKeyFile:  *tlsKey,
	}

	if err := server.Serve(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
