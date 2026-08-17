# DumbProtocol 🔐

`DumbProtocol` is a lightweight, production-ready 2FA (Two-Factor Authentication) & TOTP microservice written in Go. 

It provides a decoupled API for managing 2FA keys, rendering QR codes, validating 6-digit passcodes, and serving real-time TOTP codes — making it ideal for standard web/mobile applications as well as SMS-gateway fallbacks for feature phones ("dumbphones").

---

### 🏛️ Architecture (`internal/`)

Organized into clear operational layers:

- **`config/`**: Environment variable parsing, defaults, and validation.
- **`database/`**: SQL connection pooling (PostgreSQL & SQLite) and automated schema migrations.
- **`repository/`**: Data Access Layer (DAL) separating database queries behind clean interfaces.
- **`service/`**: Core business logic implementing RFC 6238 TOTP, QR code generation, clock drift tolerance, and SHA-256 hashed recovery codes.
- **`handler/`**: REST JSON HTTP handlers, route definitions (Chi router), and status response wrappers.

---

### 🔌 REST API Endpoints

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/healthz` | System health and status check |
| `POST` | `/api/v1/totp/setup` | Generate TOTP secret key, QR code Data URI, and initial backup codes |
| `POST` | `/api/v1/totp/verify` | Validate a 6-digit passcode or single-use recovery backup code |
| `POST` | `/api/v1/totp/code` | Fetch current live 6-digit TOTP passcode and remaining validity seconds |
| `POST` | `/api/v1/totp/recovery` | Regenerate or redeem single-use emergency backup recovery codes |

---

### 🚀 Quickstart & Makefile Commands

```bash
# Run local development server (loads .env automatically)
make dev

# Run unit and HTTP integration tests
make test

# Build Go binary into bin/dumbprotocol
make build

# Tidy and vendor dependencies
make tidy

# Create a new version tag (e.g. v0.1.0 -> v0.1.1)
make tag

# Build multi-stage Docker image
make docker
```

---
