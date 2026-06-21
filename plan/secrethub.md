# 🔐 SecretHub — Gerenciador de Secrets Pessoal

> **Problema:** Gerenciar `.env` na mão é arcaico. Infisical é pesado pro RPi (128MB+ RAM) e a cloud free estoura os limites rápido. SOPS + age é poderoso mas sem UI.
>
> **Solução:** Um servidor Go leve (~8MB RAM) com dashboard `html/template` + 2FA via Google Authenticator + vault criptografado em disco. Um dev, vários projetos, zero dependência externa.

---

## Stack

| Camada | Tecnologia | Motivo |
|---|---|---|
| **Linguagem** | Go 1.22+ | Binário único, performático, ~12MB |
| **Auth** | `golang.org/x/crypto/bcrypt` + `github.com/pquerna/otp` (TOTP) | Hash de senha + Google Authenticator |
| **Criptografia** | `golang.org/x/crypto/nacl/secretbox` | Criptografia autenticada (XChaCha20-Poly1305) |
| **Dashboard** | `html/template` + `net/http` | Zero deps, CSS inline dark mode |
| **Storage** | JSON criptografado em `~/.secrethub/` | Sem banco, sem schema, portátil |
| **2FA** | TOTP (Google Authenticator / Authy) + 10 recovery codes | Padrão, testado, offline |

---

## Arquitetura

```
~/.secrethub/
├── config.json               ← Config (porta, etc) — não criptografado
├── master.hash               ← bcrypt hash da master password
├── totp.secret               ← TOTP secret (criptografado com master password)
├── recovery.hashes           ← SHA-256 dos 10 recovery codes
└── vaults/
    ├── publify.enc           ← Vault criptografado (JSON)
    ├── alert-proxy.enc
    └── voice-pipe.enc
```

### Fluxo de inicialização:

```
secrethub serve
  │
  ├── [1ª vez] Setup wizard (CLI):
  │     ├─ Define master password
  │     ├─ Escaneia QR code no Google Authenticator
  │     └─ Salva 10 recovery codes
  │
  └── [diário] Abre dashboard em localhost:4949
        ├─ Login: master password + TOTP
        └─ CRUD de vaults + variáveis
```

### Fluxo de uso (export):

O dashboard tem um botão "Exportar .env" que copia pro clipboard ou baixa.
Ou via CLI:

```bash
secrethub export publify              # STDOUT: KEY=VALUE
secrethub export publify --dotenv     # Gera .env file
secrethub export publify --run ./api  # Exporta + executa o binário
```

---

## Especificação do Dashboard (html/template)

Uma única página SPA-like com navegação por hash (#):

### Login `/`
- Campo: Master Password
- Campo: Código TOTP (6 dígitos) — ou recovery code
- Botão: Entrar
- *Na primeira vez só: wizard de setup (QR code + recovery)*

### Home `/`
- Sidebar esquerda: lista de vaults (projetos)
- Cards de atalho: Copiar .env, Exportar, Abrir no terminal

### Vault Editor `/#/vault/{name}`
Tabela de variáveis com:
| Key (editável) | Value (editável / show/hide) | Ações |
|---|---|---|
| `DB_PASSWORD` | `••••••••••` | 👁️ 📋 🗑️ |
| `JWT_SECRET` | `••••••••••` | 👁️ 📋 🗑️ |
| | | ➕ **Nova variável** |

Botões de ação: **Salvar** (re-criptografa o vault) | **Exportar .env** | **Copiar tudo**

### Settings `/#/settings`
- Reconfigurar 2FA (novo QR code)
- Ver recovery codes (criptografado, pede senha de novo)
- Alterar master password

---

## Wireframe ASCII do Dashboard

```
┌──────────────────────────────────────────────────────┐
│  🔐 SecretHub                        [admin] [sair] │
├──────────┬───────────────────────────────────────────┤
│          │                                           │
│  📁 publify   ◄── selecionado        🔍 Exportar    │
│  📁 web       │                     📋 Copiar       │
│  📁 api       │                                     │
│  📁 alert-proxy│  ┌─ Key ────────┼─ Value ────────┬─┤
│  📁 voice-pipe │  │ DB_HOST      │ localhost      │ │
│               │  │ DB_PASSWORD  │ •••••••••••••  │ │
│  ➕ Novo       │  │ JWT_SECRET   │ •••••••••••••  │ │
│               │  │ API_KEY      │ •••••••••••••  │ │
│               │  └──────────────┴────────────────┴─┤ │
│               │                                   │ │
│               │  [➕ Nova Variável] [💾 Salvar]   │ │
│               │                                   │ │
├──────────┴───────────────────────────────────────────┤
│  🔓 Vault descriptografado na sessão (expira 15min) │
└──────────────────────────────────────────────────────┘
```

---

## Modelo de Segurança

| Risco | Mitigação |
|---|---|
| Vazar master password | 2FA TOTP obrigatório — não entra sem o código |
| Perder o celular | 10 recovery codes (mostrados 1x, papel na gaveta) |
| Reset total | Recovery codes **ou** deleta `~/.secrethub/` e recria (perde vaults) |
| Ataque no localhost | Só escuta em `127.0.0.1` — nunca exposto |
| Sessão aberta | Timeout de 15min de inatividade |

### Criptografia:
- Cada vault é criptografado com `secretbox` (XChaCha20-Poly1305)
- A chave do vault é derivada da master password + salt (argon2id)
- Recovery codes são armazenados como SHA-256 (irreversível)
- TOTP secret é criptografado com a master password

---

## CLI Reference

```bash
secrethub serve          # Inicia o servidor web
secrethub export <name>  # Exporta vault como KEY=VALUE
  --dotenv               # Gera .env file no diretório
  --run <cmd>            # Exporta + executa comando
secrethub list           # Lista vaults disponíveis
secrethub setup          # Re-executa wizard de setup
secrethub version        # Mostra versão
```

---

## Dependências Go

```
require (
    github.com/pquerna/otp v1.4.0        # TOTP
    github.com/boombuler/barcode v1.0.1  # QR code (by pquerna/otp)
    golang.org/x/crypto v0.28.0          # bcrypt + nacl/secretbox + argon2
)
```

Nada mais. **4 dependências diretas**, o resto é stdlib.

---

## Critérios de Entrega

- [ ] `go mod init github.com/publiquei/secrethub` — estrutura do projeto
- [ ] `secrethub setup` — wizard CLI de primeira execução (master password + QR code + recovery)
- [ ] `secrethub serve` — servidor HTTP com login + 2FA + sessão
- [ ] Dashboard — CRUD de vaults + variáveis (tela única, hash routing)
- [ ] Criptografia — vault .enc com secretbox, argon2id key derivation
- [ ] `secrethub export` — CLI export em 3 formatos (stdout, .env, --run)
- [ ] Timeout de sessão 15min
- [ ] Recovery codes (10 códigos, mostrar 1x, reexibir só com senha)
- [ ] Reconfigurar 2FA pelo dashboard

---

## Por que GO e não Python/Rust?

| GO | Python | Rust |
|---|---|---|
| Binário único (~12MB) | Precisa de runtime | Binário único (~3MB) |
| stdlib serve (`net/http`, `html/template`) | Flask/Django | `actix-web` ou `axum` |
| Curva baixa | Curva baixa | Curva alta |
| ~8MB RAM | ~30MB RAM | ~3MB RAM |
| **← Você já domina** | Você não usa | Você não usa |

Go é a escolha certa pra você. É na sua zona de conforto e entrega o que precisa.

---

## Links

- [[tarefas|Tarefas / Roadmap]]
- [[publiquei/produto/decisões|Decisões do Publiquei]] (inspiração de estrutura)
