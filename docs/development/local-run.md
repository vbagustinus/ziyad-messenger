# Running and Checking the Platform Locally

## Option 1: Docker (recommended)

From the **project root** (`ziyad-mesengger/`):

```bash
# Build all service images
make build
# or: docker-compose build

# Start all services in the background
make run-all
# or: docker-compose up -d

# View logs (optional)
make logs
# or: docker-compose logs -f
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
make stop-all
# or: docker-compose down
```

---

## Option 2: Run Go services directly (no Docker)

Requires **Go 1.22+** and **CGO** (for SQLite). From project root:

```bash
# Create data dirs
mkdir -p data/auth data/chat data/audit data/files data/raft

# Run each service in a separate terminal (or use a process manager).

# Terminal 1 – Auth
cd services/auth && go run . &

# Terminal 2 – Messaging
cd services/messaging && go run . &

# Terminal 3 – Discovery
cd services/discovery && go run . &

# Terminal 4 – Presence
cd services/presence && go run . &

# Terminal 5 – Audit
cd services/audit && go run . &

# Terminal 6 – File transfer
cd services/filetransfer && go run . &

# Terminal 7 – Cluster
cd services/cluster && go run . &
```

Or use the workspace and run from root (each service has its own `main`):

```bash
go run ./services/auth
go run ./services/messaging
# … etc.
```

Data is written under `data/` (and `./storage` for filetransfer if not overridden). Same `curl` commands as above; ensure ports 8081, 8083, 8084, 8086, etc. are free.

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
- **Docker build fails**: Ensure Docker context is the repo root so `pkg/` and `services/` are available; use `docker-compose build` from the root.
