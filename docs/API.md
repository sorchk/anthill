# Anthill - API Documentation

**Base URL:** `http://localhost:8080/api`

**Authentication:** Bearer token in Authorization header

---

## Auth Endpoints

### POST /api/auth/login
Login and get JWT token.

**Request:**
```json
{"username": "admin", "password": "admin123"}
```

**Response:**
```json
{"token": "eyJ...", "user": {"id": 1, "username": "admin", "role": "admin"}}
```

### POST /api/auth/logout
Logout (invalidate token).

---

## User Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/users | List users |
| POST | /api/users | Create user |
| GET | /api/users/:id | Get user |
| PUT | /api/users/:id | Update user |
| DELETE | /api/users/:id | Delete user |
| PUT | /api/users/:id/password | Change password |

---

## Node Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/nodes | List nodes |
| POST | /api/nodes | Create node |
| GET | /api/nodes/:id | Get node |
| PUT | /api/nodes/:id | Update node |
| DELETE | /api/nodes/:id | Delete node |
| POST | /api/nodes/:id/connect | Connect to node |

---

## Plugin Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/plugins | List plugins |
| POST | /api/plugins | Upload plugin |
| GET | /api/plugins/:id | Get plugin |
| DELETE | /api/plugins/:id | Delete plugin |
| GET | /api/plugins/:id/download | Download WASM |

---

## Deploy Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/deploy | List deploy tasks |
| POST | /api/deploy | Create deploy task |
| GET | /api/deploy/:id | Get deploy task |

---

## Error Responses

```json
{"error": "Error message"}
```

**Status Codes:**
- 200 - Success
- 201 - Created
- 400 - Bad Request
- 401 - Unauthorized
- 403 - Forbidden
- 404 - Not Found
- 500 - Internal Error