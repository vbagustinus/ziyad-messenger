# Running and Checking the Platform Locally

## Option 1: Docker (recommended)

From the **project root** (`ziyad-mesengger/`):

```bash
# Build all service images
make -f backend/deploy/Makefile build
# or:
cd backend/deploy && make build
# or: docker-compose -f backend/deploy/docker-compose.yml build

# Start all services in the background
make -f backend/deploy/Makefile run-all
# or:
cd backend/deploy && make run-all
# or: docker-compose -f backend/deploy/docker-compose.yml up -d

# View logs (optional)
make -f backend/deploy/Makefile logs
# or:
cd backend/deploy && make logs
# or: docker-compose -f backend/deploy/docker-compose.yml logs -f
```

### Service ports (localhost)

| Service      | Port  | Check URL / usage                    |
|-------------|-------|--------------------------------------|
| Discovery   | 8080  | UDP 5353 (mDNS)                      |
| Messaging   | 8081  | http://localhost:8081/send           |
| File Transfer | 8082 | http://localhost:8082/upload, /download |
| Presence    | 8083  | http://localhost:8083/status          |
| Audit       | 8084  | http://localhost:8084/log            |
| Cluster     | 8085  | http://localhost:8085/join           |
| Auth        | 8086  | http://localhost:8086/login, /register |

### Quick health check

```bash
# Auth health
curl -s http://localhost:8086/health

# Register a user (JSON body with username, password, role)
curl -s -X POST http://localhost:8086/register \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret","role":"admin"}'

# Login
curl -s -X POST http://localhost:8086/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"secret"}'

# Send a message (base64 content example; requires X-User-ID if your API uses it)
curl -s -X POST http://localhost:8081/send \
  -H "Content-Type: application/json" \
  -H "X-User-ID: <user_id_from_login>" \
  -d '{"channel_id":"general","content":"SGVsbG8=","type":1}'

# Presence heartbeat
curl -s -X POST http://localhost:8083/heartbeat \
  -H "Content-Type: application/json" \
  -d '{"user_id":"<user_id>","status":1}'

# Presence status
curl -s "http://localhost:8083/status?user_id=<user_id>"
```

### Stop

```bash
make -f backend/deploy/Makefile stop-all
# or:
cd backend/deploy && make stop-all
# or: docker-compose -f backend/deploy/docker-compose.yml down
```

---

## Option 2: Run Go services directly (no Docker)

Requires **Go 1.22+** and **CGO** (for SQLite). From project root:

```bash
# Create data dirs
mkdir -p backend/deploy/data/auth backend/deploy/data/chat backend/deploy/data/audit backend/deploy/data/files backend/deploy/data/raft

# Run each service in a separate terminal (or use a process manager).

# Terminal 1 – Auth
cd backend/services/auth && go run . &

# Terminal 2 – Messaging
cd backend/services/messaging && go run . &

# Terminal 3 – Discovery
cd backend/services/discovery && go run . &

# Terminal 4 – Presence
cd backend/services/presence && PRESENCE_DB_PATH=../../deploy/data/shared/platform.db go run . &

# Terminal 5 – Audit
cd backend/services/audit && go run . &

# Terminal 6 – File transfer
cd backend/services/filetransfer && go run . &

# Terminal 7 – Cluster
cd backend/services/cluster && go run . &
```

Or use the workspace and run from root (each service has its own `main`):

```bash
go run ./backend/services/auth
go run ./backend/services/messaging
# … etc.
```

Data is written under `backend/deploy/data/` (and `./storage` for filetransfer if not overridden). Same `curl` commands as above; ensure ports 8081, 8083, 8084, 8086, etc. are free.

---

## Option 3: Flutter client (login and chat)

1. **Start the backend** (Docker or native) so Auth (8086) and Messaging (8081) are running.

2. **Run the Flutter app** from the project root or from `clients/flutter_app`:

```bash
cd clients/flutter_app
flutter pub get
flutter run -d macos   # or -d windows, -d linux, -d chrome
```

3. **First time:** On the login screen, tap **"No account? Register"**, enter username and password, then **Register**. Then switch to **Login** and sign in with the same credentials.

4. **Login:** Enter the same username and password and tap **Login**. You should land on the home screen with Channels / DMs. Open a channel (e.g. **General**) and send a message; it will go to the running Messaging service with your user ID.

**URLs:** The app uses `http://localhost:8086` (Auth) and `http://localhost:8081` (Messaging) by default, which works for desktop and iOS simulator. For **Android emulator** use the host machine’s address:

```bash
flutter run -d android --dart-define=AUTH_BASE_URL=http://10.0.2.2:8086 --dart-define=MESSAGING_BASE_URL=http://10.0.2.2:8081
```

For a **physical device**, use your computer’s LAN IP (e.g. `http://192.168.1.100:8086` and `...:8081`).

---

## Troubleshooting

- **Port in use**: Change `PORT` (or the port in code) per service, or stop the process using the port.
- **Auth 404 / wrong port**: Auth defaults to `8086`; use `http://localhost:8086` for login/register.
- **Messaging “anonymous” sender**: Send `X-User-ID: <user_id>` with the request (or log in and use the token your client sends).
- **Docker build fails**: Ensure Docker context is the repo root so `backend/pkg/` and `backend/services/` are available; use `docker-compose -f backend/deploy/docker-compose.yml build` from the root.
