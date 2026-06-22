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

### 📥 .env Import

Import an existing `.env` file (or pasted content) into any vault via the web dashboard:

- **Drag-and-drop** a `.env` file directly into the dropzone inside an open vault
- **Paste** `KEY=value` lines into the textarea
- Parser tolerates `# comments`, single/double quotes, `export prefix`, empty lines, and `KEY=` without value
- Existing keys are **overwritten**, new keys are added — a toast reports the count

Import endpoint also available via API:

```bash
curl -X POST http://127.0.0.1:4949/api/vault/production/import \
  --data-binary @.env
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
- Import `.env` via drag-and-drop or paste
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

Imagem publicada no Docker Hub (multi-arch `linux/amd64` + `linux/arm64`):

```bash
# Pull direto (Raspberry Pi 3/4/5, x86_64 — detecta arquitetura automaticamente)
docker pull victorhdchagas/secrethub:latest

# Run standalone (data em ~/.secrethub-docker no host)
docker run -d --name secrethub \
  -p 127.0.0.1:4949:4949 \
  -v ${HOME}/.secrethub-docker:/home/secrethub/.secrethub \
  --restart unless-stopped \
  victorhdchagas/secrethub:latest

# Ou via compose (com Cloudflare Tunnel opcional)
docker compose -f docker/docker-compose.yml up -d
```

Tags disponíveis: `latest` (rolling) e `v0.1.0` (versão atual).

Para build local sem o pull:

```bash
docker compose -f docker/docker-compose.yml up -d --build

# Custom data location
SECRETHUB_DATA=/mnt/nas/secrethub docker compose -f docker/docker-compose.yml up -d

# Stop
docker compose -f docker/docker-compose.yml down
```

Acesse em [http://localhost:4949](http://localhost:4949) — o setup web aparece na primeira execução.

#### Data location & backup

O vault e metadados ficam em `${SECRETHUB_DATA:-~/.secrethub-docker}` no host:

```
~/.secrethub-docker/
├── master.hash          bcrypt hash
├── totp.secret          TOTP secret (encrypted)
├── recovery.hashes      SHA-256 of recovery codes
├── salt                 argon2id salt (required to decrypt vaults!)
├── machine.tokens       CI/CD tokens (encrypted JSON)
└── vaults/
    └── production.enc   Encrypted vault (XChaCha20-Poly1305)
```

**Todos os arquivos já estão cifrados ou hasheados** — safe para backup em cloud direto:

```bash
rclone copy ~/.secrethub-docker remote:secrethub-backup
restic backup ~/.secrethub-docker
```

> ⚠️ Faça backup do **diretório inteiro**. Sem o `salt`, o vault é indecifrável mesmo com a master password correta.

#### UID requirement

O container roda como UID 1000. Se seu user no host não é UID 1000:

```bash
mkdir -p ~/.secrethub-docker && sudo chown 1000:1000 ~/.secrethub-docker
```

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