# SecretHub — AGENTS.md

> **O que é:** Servidor Go leve (~8MB RAM) com dashboard html/template + 2FA (TOTP) + vault criptografado em disco.
> Substitui `.env` manual pra um dev com múltiplos projetos no RPi.
>
> Autenticado. Criptografado. 0 dependência externa.

---

## Regras Obrigatórias

### 🧪 Testes fortes
- **Todo arquivo .go** deve ter seu `_test.go` correspondente.
- Cobertura mínima: **70%** nas packages `auth`, `vault`, `server` (exceto templates).
- Testes Tabela-driven (como no Publiquei) — cenários: sucesso, erro, edge case, nil/empty.
- Testar caminhos criptográficos: decriptar com chave errada → erro, vault corrompido → erro.
- `go test -race ./...` precisa passar limpo.

### 📏 Limites de código
- **Máximo 200 linhas por arquivo `.go`** — se passar, refatore (extrair pacote, separar responsabilidade).
- **Máximo 200 caracteres por linha** — `gofmt` cuida da formatação, mas evite linhas monstro.
- **Máximo 3 níveis de indentação** — extraia pra função nomeada se precisar de mais.

### 🧹 DRY + Boas práticas Go
- stdlib primeiro, dependência externa só se não tiver alternativa viável.
- Zero `init()`. Zero variáveis globais de pacote. Dependências injetadas via struct.
- `error` sempre tratado — não engolir com `_`. Se for intencional, comentar `// intentionally discarded`.
- Funções com 3+ parâmetros do mesmo tipo (`string, string, string`) → struct nomeado.
- `context.Context` é sempre o primeiro parâmetro em funções de I/O, DB, crypto.
- Nomes em inglês (variáveis, funções, arquivos). Comentários em PT-BR onde a lógica for não-trivial.
- Arquivos nomeados pelo que contêm: `auth.go`, `vault.go`, `totp.go`, `recovery.go`.

### 🔐 Segurança (Não Negociável)
- Master password: bcrypt com cost 12+.
- Derivação de chave: argon2id (sal de 16 bytes, time=3, mem=64MB).
- Vault em disco: sempre criptografado (`secretbox`). Só decriptado em memória durante a sessão.
- TOTP secret armazenado criptografado com master password (nunca plaintext).
- Recovery codes: SHA-256 hash (nunca plaintext no disco). Mostrados 1x no setup; reexibir só após reautenticação.
- Servidor escuta **exclusivamente em `127.0.0.1`** — bind em `0.0.0.0` proibido.
- Session token: crypto/rand, 32 bytes, armazenado apenas em memória.

---

## Arquitetura

```
cmd/secrethub/main.go              ← Ponto de entrada (leve: só parse de flags + server.Serve)
internal/
├── server/
│   ├── server.go                  ← HTTP server (rotas, middlewares, bind)
│   ├── server_test.go
│   ├── handlers.go               ← Handlers do dashboard (login, vaults, settings)
│   ├── handlers_test.go
│   ├── middleware.go              ← Session + TOTP + CORS headers
│   └── middleware_test.go
├── auth/
│   ├── auth.go                    ← Interfaces + structs
│   ├── password.go                ← bcrypt hash + verify
│   ├── password_test.go
│   ├── totp.go                    ← TOTP generate + validate
│   ├── totp_test.go
│   ├── recovery.go               ← Generate + validate recovery codes
│   └── recovery_test.go
├── vault/
│   ├── vault.go                   ← Vault struct + CRUD (in-memory, decriptado)
│   ├── vault_test.go
│   ├── crypto.go                  ← secretbox encrypt/decrypt + argon2id key derivation
│   ├── crypto_test.go
│   ├── store.go                   ← Load/save .enc files em disco
│   └── store_test.go
└── templates/
    ├── embed.go                   ← //go:embed dos templates
    ├── login.html                 ← Página de login + setup wizard
    ├── dashboard.html             ← Home + vault editor + settings (navegação por hash)
    └── styles.css                 ← Dark mode inline
```

### Limites verificados:
- `auth.go`, `password.go`, `totp.go`, `recovery.go` — cada um < 200 linhas ✅
- `vault.go`, `crypto.go`, `store.go` — cada um < 200 linhas ✅
- `server.go`, `handlers.go`, `middleware.go` — cada um < 200 linhas ✅

---

## Fluxo de Dados

```
Setup (1ª execução):
  CLI wizard
    → bcrypt(master_password) → master.hash
    → totp.Generate() → print QR code
    → 10 recovery codes → SHA-256 → recovery.hashes
    → Cria ~/.secrethub/vaults/ vazio

Login:
  POST /api/login { password, totp_code }
    → bcrypt.Verify(password, master.hash)
    → totp.Validate(totp_code, totp_secret)  # ou recovery.Validate()
    → Gera session token (32 bytes crypto/rand)
    → Deriva vault key: argon2id(password, salt) → 32 bytes
    → Descriptografa vaults com a vault key → mantém em memória na sessão

CRUD de variáveis:
  GET/POST/PUT/DELETE /api/vault/{name}/keys/{key}
    → [middleware] verifica session token
    → Lê/escreve no vault em memória (decryptado)
    → Ao salvar: re-criptografa com vault key → write .enc

Export:
  GET /api/vault/{name}/export
    → [middleware] verifica session token
    → Serializa vault como KEY=VALUE\n
    → Retorna texto plano (Content-Type: text/plain)

CLI:
  secrethub export <name>
    → Lê stdin: master password (prompt)
    → Deriva vault key: argon2id(password, salt)
    → Decripta vault
    → Printa KEY=VALUE
```

---

## Testing Strategy

| Package | O que testar |
|---|---|
| **auth/password** | Hash + verify (senha correta, errada, empty) — cost mínimo no teste (bcrypt cost=4 pra não travar CI) |
| **auth/totp** | Generate secret → Validate com código válido, código expirado, código de outro secret, código vazio |
| **auth/recovery** | Generate 10 códigos → Validate cada um → Re-validate (deve falhar) — cada código é one-time |
| **vault/vault** | CRUD: set/get/delete, get de key inexistente, overwrite, empty values |
| **vault/crypto** | Encrypt → Decrypt (dados originais), Decrypt com chave errada → erro, Decrypt com dados corrompidos → erro |
| **vault/store** | Save → Load (round-trip), Load de arquivo inexistente → erro, Save em diretório sem permissão → erro |
| **server/handlers** | Login (sucesso, senha errada, TOTP errado), CRUD vault (autenticado, não autenticado), Export |
| **server/middleware** | Rota protegida sem cookie → 401, Rota protegida com cookie inválido → 401, Rota protegida com cookie válido → 200 |

### Setup de teste seguro:
- `bcrypt` cost = **4** em testes (não 12) — não trava o CI.
- Temp dir em `t.TempDir()` pra simular `~/.secrethub/`.
- Testes paralelos com `t.Parallel()` onde não houver race condition.

---

## Build & Run

```bash
# Desenvolvimento (no PC)
go build -o secrethub ./cmd/secrethub

# Cross-compile pro RPi (ARM64)
GOOS=linux GOARCH=arm64 go build -o secrethub-arm64 ./cmd/secrethub

# Testes
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Lint (se disponível)
golangci-lint run ./...
```

---

## Ordem de Implementação (6 Prompts)

| Prompt | Arquivos | ~Linhas |
|---|---|---|
| **1** Setup + CLI skeleton + `cmd/secrethub/main.go` + `go.mod` | `cmd/secrethub/main.go`, `internal/server/server.go` esqueleto | ~100 |
| **2** Auth: `auth/password.go` + `auth/totp.go` + `auth/recovery.go` + wizard CLI | 6 arquivos (3 fonte + 3 teste) | ~250 |
| **3** Vault: `vault/vault.go` + `vault/crypto.go` + `vault/store.go` + testes | 6 arquivos (3 fonte + 3 teste) | ~250 |
| **4** Dashboard: `server/handlers.go` + `server/middleware.go` + templates HTML | `handlers.go`, `middleware.go`, `login.html`, `dashboard.html` | ~400 |
| **5** CLI export: `secrethub export` (stdout, --dotenv, --run) | Extensão do `main.go` + `vault/export.go` | ~150 |
| **6** Polish: timeout 15min, CSS, cross-compile, `golangci-lint` | Ajustes nos arquivos existentes | ~100 |
| **🧪** Testes: escritos JUNTO com cada prompt acima (não deixar pro final) | `_test.go` em cada package | ~400 |

---

## Lembretes pro Agente de Código

- **Testes primeiro?** Não — aqui o fluxo é código + testes no mesmo prompt (mas testes não são opcionais).
- **DRY:** Se repetiu `encryptVault` / `decryptVault` em mais de 2 lugares, extrai pra `vault/crypto.go`.
- **200 linhas:** Se um arquivo `.go` passar de 200 linhas, pare e refatore antes de continuar.
- **html/template** não é React — aceite as limitações. CSS inline, navegação por hash (`#/vault/xyz`).
- **Não usar** `gorilla/mux`, `chi`, `gin` — `net/http` + `http.ServeMux` (Go 1.22+ com pattern `GET /api/vault/{name}`) resolve.
