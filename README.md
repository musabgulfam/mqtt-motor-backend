# Go MQTT Backend

A backend server written in Go that communicates with an ESP32 device via MQTT and manages users with authentication. It also implements a queue and daily quota for motor-on requests.

---

## Features
- User registration and login with JWT authentication (registration is now email/password only)
- MQTT communication with ESP32 (publish/subscribe)
- REST API for user and device interaction
- Queueing and daily quota for motor-on requests
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

### 6. Run the Server
```sh
go run main.go
```
The server will start on `http://localhost:8080`.

### 7. Run Automated Tests
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
   |--[Motor Queue & Quota Logic]
```

- **Gin**: Web framework for REST API
- **GORM**: ORM for SQLite database
- **Paho MQTT**: MQTT client for Go
- **JWT**: Authentication for protected endpoints

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
│ created_at      │    │ duration            │    │ timestamp       │
│ updated_at      │    │ status              │    │ topic           │
└─────────────────┘    └─────────────────────┘    └─────────────────┘
```

**Relationships:**
- `users` → `motor_requests` (One-to-Many)
- Future: `users` → `device_data` (One-to-Many)

**Notes:**
- Currently only `users` table exists
- Motor requests are handled in-memory (queue system)
- Device data could be stored in database for persistence
- All timestamps use SQLite's built-in datetime functions

---

## Project Structure

```
go-mqtt-backend/
├── main.go              # Entry point - orchestrates everything
├── go.mod/go.sum        # Go module dependencies
├── data.db              # SQLite database (auto-generated)
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
│   └── user_test.go     # Automated tests for user handlers
├── middleware/
│   └── auth.go          # JWT authentication middleware
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

---

## Motor Queue & Quota Logic
- All motor-on requests are queued.
- Each request specifies a duration.
- If the total requested time in 24h exceeds the quota, further requests are rejected until the quota resets.
- Actual motor control logic is commented out for safety.

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

---

## Extending
- Add more device endpoints or logic in `handlers/mqtt.go`
- Store device data in the database
- Add user roles or admin features
- Switch to PostgreSQL/MySQL by changing the GORM driver

---

## License
MIT 