# Expire Share

A file sharing service with expiration and download limits. Upload files with optional password protection, TTL, and a maximum number of downloads. Integrates with a separate [auth-service](https://github.com/mkaascs/AuthService) for user authentication via gRPC.

---

## Features

- **File upload** — multipart/form-data with configurable TTL and download limit
- **Password protection** — optional bcrypt-hashed password per file
- **Auto-deletion** — file is automatically deleted after the last download or when TTL expires
- **Access control** — only the file owner can delete or view file info
- **JWT authentication** — token validation delegated to auth-service via gRPC
- **Role-based upload limits** — regular users have a configurable upload cap; VIP users get a higher limit
- **Clean architecture** — domain-driven design with clear separation of handlers, services, and repositories

---

## Tech Stack

| Component | Technology                                                            |
|-----------|-----------------------------------------------------------------------|
| **Language** | Go 1.24+                                                              |
| **HTTP Router** | chi                                                                   |
| **Auth** | JWT via [auth-service](https://github.com/mkaascs/AuthService) (gRPC) |
| **Database** | MySQL 8.0                                                             |
| **File Storage** | Local filesystem                                                      |
| **Logging** | slog (structured JSON/text)                                           |
| **Migrations** | golang-migrate                                                        |
| **Documentation** | Swagger (swaggo)                                                      |

---

## API

Full Swagger documentation available at `/swagger/index.html` when running locally.

### Auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/auth/register` | Register a new user |
| `POST` | `/api/auth/login` | Login and receive token pair |
| `POST` | `/api/auth/refresh` | Refresh access token |
| `POST` | `/api/auth/logout` | Invalidate tokens |

### Files

| Method | Endpoint | Auth | Description                                         |
|--------|----------|------|-----------------------------------------------------|
| `POST` | `/api/upload` | Required | Upload a file                                       |
| `GET` | `/api/file` | Required | Get all user file info (downloads left, expires in) |
| `GET` | `/api/file/{alias}` | Required | Get file info (downloads left, expires in)          |
| `DELETE` | `/api/file/{alias}` | Required | Delete a file                                       |
| `GET` | `/download/{alias}` | — | Download a file                                     |

### Admin

| Method | Endpoint                             | Auth | Description           |
|--------|--------------------------------------|------|-----------------------|
| `GET`  | `/api/admin/users`                   | Required | Get all users info    |
| `GET`  | `/api/admin/users/{id}`              | Required | Get user info by ID   |
| `POST` | `/api/admin/users/{id}/roles/assign` | Required | Assign a role to user |
| `POST` | `/api/admin/users/{id}/roles/revoke` | Required | Revoke a role of user |

#### Upload request (multipart/form-data)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | file | Yes | File to upload |
| `ttl` | string | No | Time to live, e.g. `1h`, `2h30m`, `7d`. Default from config |
| `max_downloads` | int | No | Max download count (1–10000). Default from config |
| `password` | string | No | Password to protect the file |

Password-protected files require the `X-Resource-Password` header on download and delete.

---

## Quick Start

### Prerequisites

- Go 1.24+
- Docker + Docker Compose
- [auth-service](https://github.com/mkaascs/AuthService) running (provides JWT validation)
- Task (optional)

### Environment variables

| Variable | Description              | Required |
|----------|--------------------------|----------|
| `CONFIG_PATH` | Path to config file      | Yes |
| `MYSQL_ROOT_PASSWORD` | MySQL root password      | Yes |
| `CORS_ALLOWED_ORIGINS` | Allowed origins for CORS | Yes |
| `ADMIN_SECRET_BASE64` | Base64 secret for admins | Yes |

### Config file (config/dev.yaml)

```yaml
env: "dev" # prod, local, dev
db_host: "mysql-expire:3306"
storage:
  type: "local"
  path: "./storage/"
  max_file_size: "500mb"
http_server:
  port: 8080
  timeout: 4s
  idle_timeout: 60s
  cors:
    allowed_credentials: true
    max_age: 86400 # 24h
service:
  default_ttl: 1h
  default_max_downloads: 1
  alias_length: 6
  file_worker_delay: 5m
  permissions:
    max_uploaded_files_for_vip: 10
    max_uploaded_files_for_user: 1
auth_service:
  addr: "auth-service:5505"

```
---
## Docker networking

expire-share and auth-service communicate over a shared Docker network `services-network`. auth-service must be started first — it creates the network. expire-share connects to it as an external network.

```
auth-service container  ──gRPC──►  expire-share container
     └── services-network (shared)
```

---

## Security

- Passwords are stored as bcrypt hashes — never in plain text
- File access requires matching `user_id` — other users get `403 Forbidden`
- Admin role bypasses all ownership and password checks
- JWT validation is stateless — delegated entirely to auth-service
- Token is passed via `Authorization: Bearer <token>` header