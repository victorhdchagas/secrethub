# SecretHub 🔐

**Self-hosted, encrypted secrets manager for solo developers.**

SecretHub replaces `.env` files scattered across your projects with a lightweight web dashboard + CLI — all running on `127.0.0.1`, zero cloud dependencies.

- **~8MB RAM** — runs on a Raspberry Pi Zero 2W
- **No external services** — SQLite-free, cloud-free, DNS-free
- **2FA (TOTP)** — protect your vault with Google Authenticator / Authy
- **Encrypted at rest** — argon2id + NaCl secretbox
- **CI/CD ready** — machine tokens for pipelines

---

## Quick Start

```bash
# Build from source
git clone https://github.com/victorhdchagas/secrethub
cd secrethub
make build

# Run the setup wizard
./bin/secrethub setup

# Start the server
./bin/secrethub serve
# → http://127.0.0.1:4949
```

First visit guides you through:
1. **Master password** — minimum 4 characters
2. **2FA setup** — scan QR code with your authenticator app
3. **Recovery codes** — 10 one-time use codes

---

## Features

### 🔒 Security

| Layer | Algorithm | Purpose |
|---|---|---|
| Password hashing | **bcrypt** (cost 12) | Master password verification |
| Key derivation | **argon2id** (time=3, mem=64MB) | Vault encryption key |
| Vault encryption | **NaCl secretbox** (XChaCha20-Poly1305) | Data at rest |
| Token storage | **SHA-256** | Recovery codes + machine tokens |
| Session tokens | **crypto/rand** 32 bytes | In-memory only, 15min TTL |

### 📋 Vault CRUD

Add, edit, delete keys per vault — all through the web dashboard or CLI.

```
secrethub export production
# DB_HOST=localhost
# DB_PORT=5432
# API_KEY=sk-...

secrethub export production --dotenv
# Writes .env.production

secrethub export production --run "npm run deploy"
# Sets vault vars as environment variables
```

### 🔑 Machine Tokens (CI/CD)

Generate tokens with scoped access for automated pipelines:

```
secrethub token create
```

Then in CI:

```bash
curl http://127.0.0.1:4949/api/vault/production/export?token=$SECRETHUB_TOKEN
```

Rate-limited to 30 requests/minute per token. Revocable individually from the dashboard or CLI.

### 🌐 Web Dashboard

Built with Alpine.js — no build step, no bundler. Dark mode by default.

- Real-time vault editing
- Copy individual keys or entire vault
- Export as `.env` download
- Session auto-refresh with visible expiry in Brazil time
- Machine token management in Settings

### 🏠 Architecture

```
~/.secrethub/
├── master.hash          bcrypt hash
├── totp.secret          TOTP shared secret (encrypted with vault key)
├── recovery.hashes      SHA-256 of recovery codes
├── salt                 argon2id salt (16 bytes)
├── machine.tokens       CI/CD tokens (encrypted JSON)
└── vaults/
    ├── production.enc   Encrypted vault (secretbox)
    ├── staging.enc
    └── ...
```

---

## Commands

```
secrethub serve [--port 4949] [--host 127.0.0.1] [--tls-cert file --tls-key file]
                            Start web server (default :4949)
secrethub setup              CLI setup wizard
secrethub export <name>      Export vault as KEY=VALUE
secrethub export <name> --dotenv   Write .env file
secrethub export <name> --run <cmd>   Execute with vault env
secrethub list               List available vaults
secrethub token create       Generate a CI/CD token
secrethub token revoke <p>   Revoke token by prefix
secrethub token list         List active tokens
secrethub version            Show version
```

---

## Build & Deploy

```bash
# Development
make build        # → bin/secrethub

# Raspberry Pi (ARM64)
make build-arm64  # → bin/secrethub-arm64

# Run tests
make test

# Lint
make lint
```

### systemd (RPi)

```ini
[Unit]
Description=SecretHub
After=network.target

[Service]
ExecStart=/home/pi/secrethub serve --port 4949
Restart=always
User=pi

[Install]
WantedBy=multi-user.target
```

### TLS (experimental, use with caution)

HTTPS via `--tls-cert` and `--tls-key`:

```bash
secrethub serve --tls-cert /etc/letsencrypt/live/example.com/fullchain.pem \
                --tls-key /etc/letsencrypt/live/example.com/privkey.pem
```

> ⚠️ **Security notice:** TLS support is experimental and needs refinement. The server does not enforce HTTPS redirects, HSTS headers, or certificate validation best practices. Only use behind a production reverse proxy (Caddy, Nginx, Traefik) that handles TLS termination properly. The built-in TLS is intended for testing and LAN-only use.

### Docker

```bash
# Build and start
docker compose -f docker/docker-compose.yml up -d

# Quick disposable test (temp data dir, cleans up on Ctrl+C)
mkdir -p /tmp/secrethub-test
docker compose -f docker/docker-compose.yml run --rm \
  -v /tmp/secrethub-test:/root/.secrethub \
  -p 4949:4949 \
  secrethub

# Stop compose services
docker compose -f docker/docker-compose.yml down

# Multi-arch build
docker buildx build --platform linux/amd64,linux/arm64 \
  -f docker/Dockerfile -t secrethub .
```

Acesse em [http://localhost:4949](http://localhost:4949) — o setup web aparece na primeira execução.

---

## Libraries

| Library | Purpose |
|---|---|
| [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) | argon2id, bcrypt, NaCl secretbox |
| [github.com/pquerna/otp](https://github.com/pquerna/otp) | TOTP generation & validation |
| [github.com/boombuler/barcode](https://github.com/boombuler/barcode) | QR code rendering |
| [Alpine.js](https://alpinejs.dev) (embedded) | Reactive dashboard UI |
| stdlib `net/http` | HTTP server (Go 1.22+ routing) |

---

## Why not .env?

.env files work until you need to:
- Share secrets across projects without duplication
- Rotate a credential in one place
- Keep secrets out of git by default (not by discipline)
- Access secrets from a phone or another machine on your LAN

SecretHub keeps one source of truth per vault, encrypted on disk, accessible via web UI, CLI, and HTTP API.

---

*Built with Go 1.22+. Licensed under the MIT License.*
