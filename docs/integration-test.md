# Integration Test Guide

## Prerequisites
- Docker and docker-compose installed
- Go 1.22+ for building components

## Quick Start

### 1. Start Admin Panel
```bash
docker compose up -d admin
```

### 2. Verify Admin API
```bash
curl http://localhost:8080/api/stats/health
```

### 3. Login
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

## Project Structure

```
/home/sorc/test/test2/
├── admin/              # Admin Panel Backend (Go+Gin)
│   ├── cmd/server/     # Main entry point
│   └── internal/       # Handlers, models, middleware
├── web/                # Admin Panel Frontend (Vue3+NaiveUI)
├── runtime/            # Runtime Node (if exists)
├── docker-compose.yml  # Docker deployment
└── Makefile           # Build commands
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /api/auth/login | User login |
| GET | /api/me | Get current user |
| GET | /api/stats | Dashboard stats |
| GET | /api/nodes | List nodes |
| POST | /api/nodes | Create node |
| POST | /api/nodes/:id/connect | Connect to node |
| GET | /api/plugins | List plugins |
| POST | /api/plugins | Upload plugin |
| GET | /api/users | List users (admin) |
| POST | /api/users | Create user (admin) |
| GET | /api/audit | List audit logs |
| GET | /api/deployments | List deployments |
| POST | /api/deployments | Create deployment |
| GET | /api/sessions | List sessions |
| DELETE | /api/sessions/:id | Revoke session |

## Testing with curl

See `scripts/test-api.sh` for full API test script.