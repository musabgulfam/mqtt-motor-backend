# Go MQTT Backend

A backend server written in Go that communicates with an ESP32 device via MQTT and manages users with authentication. It also implements a queue and daily quota for motor-on requests with admin force shutdown capabilities.

---

## Features
- User registration and login with JWT authentication (registration is now email/password only)
- MQTT communication with ESP32 (publish/subscribe)
- REST API for user and device interaction
- Queueing and daily quota for motor-on requests
- **Admin force shutdown and restart capabilities**
- **Role-based access control (user/admin)**
- SQLite database via GORM ORM
- Configurable via environment variables

---

## Installation & Setup

### 1. Prerequisites
- **Go** (1.18 or newer): [Install Go](https://go.dev/doc/install)
- **Mosquitto MQTT Broker** (for local MQTT):
  - **macOS:**
    ```sh
    brew install mosquitto
    brew services start mosquitto
    ```
  - **Ubuntu/Debian:**
    ```sh
    sudo apt update
    sudo apt install mosquitto mosquitto-clients
    sudo systemctl start mosquitto
    ```
  - **Windows:**
    - Download from [mosquitto.org/download](https://mosquitto.org/download/)
    - Run the installer and start Mosquitto from the Start menu or command prompt.

### 2. Clone the Repository
```sh
git clone <repo-url>
cd go-mqtt-backend
```

### 3. Install Go Dependencies
```sh
go mod tidy
```

### 4. (Optional) Install Test Dependencies
```sh
go get github.com/stretchr/testify
```

### 5. Set Environment Variables (Optional)
You can override defaults by setting environment variables:
- `DB_PATH` (default: `data.db`)
- `MQTT_BROKER` (default: `tcp://localhost:1883`)
- `JWT_SECRET` (default: `supersecret`)

Example:
```sh
export MQTT_BROKER="tcp://test.mosquitto.org:1883"
export JWT_SECRET="mysecret"
export DB_PATH="mydb.db"
```

### 6. Create Admin User (Optional)
To test admin functionality, you can configure admin creation via environment variables:

```sh
# Option 1: Use default admin (admin@example.com / admin123)
export CREATE_ADMIN=true

# Option 2: Custom admin credentials
export CREATE_ADMIN=true
export ADMIN_EMAIL="your-admin@example.com"
export ADMIN_PASSWORD="your-secure-password"

# Option 3: Disable admin creation (for production)
export CREATE_ADMIN=false
```

**Note:** Admin user is automatically created on first run if `CREATE_ADMIN=true` and no admin exists.

### 7. Run the Server
```sh
go run main.go
```
The server will start on `http://localhost:8080`.

**Note:** The database file (`data.db`) will be automatically created on first run. This file is excluded from the repository for security reasons.

### 8. Run Automated Tests
```sh
go test ./...
```
This will run all test files, including the detailed user registration/login tests.

---

## Architecture

```
Client (Postman, Web, Mobile)
   |
   | HTTP (REST API)
   v
[Go Gin Web Server]
   |--[User Handlers] <-> [SQLite DB (GORM)]
   |--[MQTT Handlers] <-> [MQTT Broker] <-> [ESP32]
   |--[Auth Middleware (JWT)]
   |--[Admin Middleware (Role-based)]
   |--[Motor Queue & Quota Logic]
   |--[Admin Shutdown Control]
```

- **Gin**: Web framework for REST API
- **GORM**: ORM for SQLite database
- **Paho MQTT**: MQTT client for Go
- **JWT**: Authentication for protected endpoints
- **Role-based Access**: Admin/user role system

---

## Database Schema (ERD)

### Current Schema
```
┌─────────────────┐
│      users      │
├─────────────────┤
│ id (PK)         │ ← Primary Key
│ email (UNIQUE)  │ ← Unique email
│ password        │ ← Hashed password
│ role            │ ← User role (user/admin)
└─────────────────┘
```

### Potential Future Schema
```
┌─────────────────┐    ┌─────────────────────┐    ┌─────────────────┐
│      users      │    │   motor_requests    │    │   device_data   │
├─────────────────┤    ├─────────────────────┤    ├─────────────────┤
│ id (PK)         │◄───│ id (PK)             │    │ id (PK)         │
│ email (UNIQUE)  │    │ user_id (FK)        │    │ device_id       │
│ password        │    │ request_at          │    │ sensor_value    │
│ role            │    │ duration            │    │ timestamp       │
│ created_at      │    │ status              │    │ topic           │
│ updated_at      │    └─────────────────────┘    └─────────────────┘
└─────────────────┘
```

**Relationships:**
- `users` → `motor_requests` (One-to-Many)
- Future: `users` → `device_data` (One-to-Many)

**Notes:**
- Currently only `users` table exists with role support
- Motor requests are handled in-memory (queue system)
- Device data could be stored in database for persistence
- All timestamps use SQLite's built-in datetime functions

---

## Project Structure

```
go-mqtt-backend/
├── main.go              # Entry point - orchestrates everything
├── go.mod/go.sum        # Go module dependencies
├── data.db              # SQLite database (auto-generated, not in repo)
├── README.md            # Documentation
├── config/
│   └── config.go        # Configuration management
├── database/
│   └── database.go      # Database connection & setup
├── models/
│   └── user.go          # Data structures (User model)
├── handlers/
│   ├── user.go          # User registration/login logic
│   ├── mqtt.go          # MQTT commands & motor queue logic
│   ├── user_test.go     # Automated tests for user handlers
│   └── admin_test.go    # Automated tests for admin functionality
├── middleware/
│   └── auth.go          # JWT authentication & admin middleware
└── mqtt/
    └── client.go        # MQTT client wrapper
```

---

## API Endpoints

### **User Management**
- `POST /register` — Register a new user
  - `{ "email": "mail", "password": "pass" }`
- `POST /login` — Login and receive JWT
  - `{ "email": "mail", "password": "pass" }`

### **Protected Endpoints** (require `Authorization: Bearer <token>`)
- `POST /api/send` — Send a command to ESP32 via MQTT
  - `{ "topic": "esp32/command", "payload": "on" }`
- `GET /api/device` — Get device data (placeholder)
- `POST /api/motor/on` — Queue a motor-on request with duration (seconds)
  - `{ "duration": 60 }`
  - Enforces a daily quota (e.g., 1 hour per 24h)
- `GET /api/motor/status` — Get system status (quota, shutdown state, queue length)

### **Admin Endpoints** (require admin role + `Authorization: Bearer <token>`)
- `POST /api/admin/shutdown` — Force shutdown motor system
  - `{ "reason": "Emergency maintenance" }`
- `POST /api/admin/restart` — Restart motor system
  - No body required

---

## Motor Queue & Quota Logic
- All motor-on requests are queued.
- Each request specifies a duration.
- If the total requested time in 24h exceeds the quota, further requests are rejected until the quota resets.
- **Admin force shutdown immediately stops all motor operations and prevents new requests.**
- **Admin restart resumes normal operation.**
- Actual motor control logic is commented out for safety.

---

## Admin Force Shutdown Algorithm

The admin force shutdown feature adds an additional layer of control over the motor queue system:

### **Shutdown State Management:**
- **Global shutdown flag:** `isShutdown` boolean
- **Shutdown metadata:** reason, admin user, timestamp
- **Thread-safe access:** Uses mutex for concurrent safety

### **Shutdown Process:**
1. **Immediate motor stop:** Sends "off" command to motor via MQTT
2. **Queue processing halt:** Background processor skips all requests during shutdown
3. **Request rejection:** New motor requests return 503 Service Unavailable
4. **State persistence:** Shutdown state maintained until admin restart

### **Restart Process:**
1. **Clear shutdown state:** Reset all shutdown flags and metadata
2. **Resume queue processing:** Background processor resumes normal operation
3. **Accept new requests:** Motor requests are processed normally again

### **Integration with Queue Algorithm:**
```
Original Queue Flow:
Request → Quota Check → Enqueue → Process → Motor Control

With Admin Shutdown:
Request → Shutdown Check → Quota Check → Enqueue → Process → Motor Control
                ↓
        503 if shutdown
```

---

## Development & Testing

### Automated Tests
- Place test files as `*_test.go` (see `handlers/user_test.go` for example)
- Run tests:
  ```sh
  go test ./...
  ```

### Linting & Formatting
```sh
go fmt ./...
go vet ./...
```

### API Testing
- Use [Postman](https://www.postman.com/), [Insomnia](https://insomnia.rest/), or `curl` to test endpoints.

---

## Example Usage

**Register:**
```sh
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass"}'
```

**Login:**
```sh
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass"}'
```

**Queue Motor Request:**
```sh
curl -X POST http://localhost:8080/api/motor/on \
  -H "Authorization: Bearer <JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"duration":60}'
```

**Get System Status:**
```sh
curl -X GET http://localhost:8080/api/motor/status \
  -H "Authorization: Bearer <JWT_TOKEN>"
```

**Admin Force Shutdown:**
```sh
curl -X POST http://localhost:8080/api/admin/shutdown \
  -H "Authorization: Bearer <ADMIN_JWT_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"reason":"Emergency maintenance"}'
```

**Admin Restart:**
```sh
curl -X POST http://localhost:8080/api/admin/restart \
  -H "Authorization: Bearer <ADMIN_JWT_TOKEN>"
```

---

## Extending
- Add more device endpoints or logic in `handlers/mqtt.go`
- Store device data in the database
- Add more admin features (quota management, user management)
- Switch to PostgreSQL/MySQL by changing the GORM driver

---

## Git Commit Standards

This project follows [Conventional Commits](https://www.conventionalcommits.org/) for clear and structured commit messages.

### Commit Message Format
```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

### Examples
```sh
# New feature
git commit -m "feat: add motor queue quota system"

# Bug fix
git commit -m "fix: resolve JWT token validation issue"

# Documentation
git commit -m "docs: add comprehensive ERD and database schema"

# Admin feature
git commit -m "feat: add admin force shutdown capability"
``` 