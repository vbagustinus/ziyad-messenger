# Admin Dashboard

Enterprise Admin Control Panel for the LAN Communication Platform. Next.js 14 (App Router), TypeScript, Tailwind, Zustand, Axios, Recharts, WebSocket.

## Setup

```bash
npm install
cp .env.example .env.local
```

## Configure

- `NEXT_PUBLIC_ADMIN_API`: Admin API base URL (default `http://localhost:8090`).

## Run

```bash
npm run dev
```

Open http://localhost:3000.

---

## How to log in

1. **Start the admin backend** (from project root):
   ```bash
   go run ./admin-service/cmd/server
   ```
   Or: `make admin-run`

2. **First-time only:** On first run the backend creates a default admin if the DB is empty:
   - **Username:** `admin`
   - **Password:** `admin`

3. **Open the dashboard:** http://localhost:3000 → you’ll be sent to the login page.

4. **Sign in** with `admin` / `admin` (or another admin account you’ve created).

---

## Create another admin user

**Option A – From the dashboard (Settings)**  
After logging in, go to **Settings**. Use the “Create admin” form to add another admin (username, password, role).

**Option B – With curl (when already logged in)**  
Get a token by logging in, then:

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8090/admin/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' | jq -r '.token')

# Create another admin
curl -s -X POST http://localhost:8090/admin/admins \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"username":"operator","password":"your-secure-password","role":"admin"}'
```

**Option B (no jq):**  
Login in the browser, then in DevTools → Application → Local Storage copy the value of `admin_token` and use it as `Authorization: Bearer <token>` in the create-admin request.

---

## Run backend

From project root:

```bash
go run ./admin-service/cmd/server
```

Or: `make admin-run`  

Default port **8090**. Data is stored in `data/admin.db` (create `data` if needed).
