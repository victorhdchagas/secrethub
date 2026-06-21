# 📋 SecretHub — Tarefas

> Progresso estimado: 0% | ⏱️ ~2h com DeepSeek no opencode free
>
> **Fluxo:** Cada tarefa = 1 prompt pro agente de código.
> Começar por **Setup** → **Auth** → **Criptografia** → **Dashboard** → **Export**

---

## ✅ Setup (estrutura)

- [ ] `go mod init github.com/publiquei/secrethub`
- [ ] Estrutura de diretórios:
  ```
  ├── cmd/secrethub/main.go
  ├── internal/
  │   ├── server/     (HTTP server + rotas)
  │   ├── auth/       (password + TOTP + recovery)
  │   ├── vault/      (CRUD + criptografia)
  │   └── templates/  (html/templates embedados)
  └── go.mod
  ```
- [ ] `go mod tidy` + build (`go build ./cmd/secrethub`)

## 🔐 Auth + Setup Wizard

- [ ] Master password: bcrypt hash, salvar em `~/.secrethub/master.hash`
- [ ] TOTP: `github.com/pquerna/otp` gerar secret + QR code no terminal
- [ ] Recovery: 10 códigos aleatórios, SHA-256 hash, mostrar no terminal
- [ ] Status do setup: se `~/.secrethub/` não existe, entra em wizard mode
- [ ] Login endpoint: POST `/api/login` — password + TOTP → session token
- [ ] Session middleware: cookie com token, 15min expiry

## 🔒 Vault Criptografia

- [ ] `golang.org/x/crypto/nacl/secretbox` — encriptar/decriptar JSON
- [ ] Key derivation: argon2id (master password + salt) → 32-byte key
- [ ] Vault CRUD interno:
  - [ ] Criar vault (JSON vazio `{}`)
  - [ ] Get variável (key → value)
  - [ ] Set variável (key: value)
  - [ ] Delete variável
  - [ ] Listar vaults
- [ ] Persistência: `~/.secrethub/vaults/{name}.enc`
- [ ] Lock automático ao decriptar (vault só fica decriptado em memória na sessão)

## 🖥️ Dashboard (html/template)

- [ ] Página de login (bcrypt + TOTP)
- [ ] Página de setup (1ª vez — QR code + recovery codes)
- [ ] Home com sidebar de vaults + atalhos
- [ ] Vault editor:
  - [ ] Tabela key/value com show/hide
  - [ ] ➕ Adicionar variável (linha nova)
  - [ ] ✏️ Editar in-place
  - [ ] 🗑️ Deletar
  - [ ] 💾 Salvar (re-criptografa)
- [ ] Botão "Copiar .env" (clipboard via JS)
- [ ] Settings:
  - [ ] Reconfigurar 2FA
  - [ ] Ver recovery codes (pede senha de novo)
- [ ] Timeout modal (15min)
- [ ] CSS dark mode inline (no template, sem deps)

## 🖨️ CLI Export

- [ ] `secrethub export <name>` — stdout `KEY=VALUE`
- [ ] `secrethub export <name> --dotenv` — gera `.env` no cwd
- [ ] `secrethub export <name> --run <cmd>` — executa comando com vars no env
- [ ] `secrethub list` — lista vaults
- [ ] `secrethub version` — mostra versão

## 📦 Empacotamento

- [ ] `//go:embed internal/templates/*` — templates compilados no binário
- [ ] Cross-compile pro RPi:
  ```bash
  GOOS=linux GOARCH=arm64 go build -o secrethub ./cmd/secrethub
  ```
- [ ] Testar: `secrethub serve` rodando no RPi, acessar `localhost:4949`

## 🔑 Machine Token (CI/CD)

> Permite que o CI/CD (GitHub Actions) acesse o vault remotamente sem navegador.
> Útil pra deploy na Oracle VPS: CI puxa o `.env` do RPi via HTTPS e manda pra produção.

- [ ] `~/.secrethub/machine.tokens` — arquivo com tokens bcrypt (hash, label, scopo)
- [ ] CLI: `secrethub token create <label>` — gera token + mostra uma vez
- [ ] CLI: `secrethub token revoke <label>` — remove token
- [ ] Endpoint: `GET /api/export/{vault}?token=<token>` — exporta vault via HTTP
  - [ ] Valida token (bcrypt.Verify contra machine.tokens)
  - [ ] Retorna `KEY=VALUE\n` (text/plain)
- [ ] Endpoint protegido: só funciona se token for válido (não precisa de sessão)
- [ ] Rate limit: 10 req/min por token (prevenir abuso se exposto)
- [ ] Documentar no AGENTS.md o fluxo de CI/CD com Machine Token

---

## Ordem recomendada de prompts

| Prompt # | Entrega | ~Linhas |
|---|---|---|
| 1 | Setup + estrutura + CLI skeleton + `secrethub version` | ~100 |
| 2 | Auth: master password + TOTP + recovery + wizard CLI | ~250 |
| 3 | Vault: criptografia secretbox + argon2id + CRUD | ~250 |
| 4 | Dashboard: login + home + vault editor + settings | ~400 |
| 5 | CLI export: stdout + --dotenv + --run | ~150 |
| 6 | Polish: timeout, CSS, cross-compile, testes | ~100 |
| **7** | **Machine Token: token create/revoke + endpoint /api/export/{vault}?token=** | **~120** |

---

## Links

- [[secrethub]] — Documentação completa do projeto
- [[publiquei/bugs/abertos]] — Bug tracker do Publiquei (referência de estrutura)
- [[publiquei/produto/mvp-roadmap]] — Exemplo de roadmap visual
