# WebSocket Chat

A small real-time chat demo: a Go WebSocket server, a React web UI, a Fyne desktop client, and PostgreSQL. The server connects to Postgres on startup (ping only; it does not read or write chat data).

Traffic from the host goes through a single **nginx** entry point on port **8080**. **web** serves the React build; **server** handles WebSocket and REST API.

## Architecture

```
Browser / Fyne  →  localhost:8080
                        │
                        ▼
                   nginx (public)
                        ├── /ws, /api/*  →  server:8080  (internal)
                        └── /*           →  web:80      (internal)

  db ← migrate (one-shot)     server → DATABASE_URL → db
```

| Service  | Role                                      | Exposed on host |
|----------|-------------------------------------------|-----------------|
| `nginx`  | Reverse proxy, only public entry          | `8080`          |
| `web`    | Vite build + static SPA (`try_files`)     | no              |
| `server` | Go: `/ws`, `/api/...`, Postgres ping      | no              |
| `db`     | PostgreSQL 16                             | no              |
| `migrate`| golang-migrate `up` before server starts  | no              |

## Project structure

```
chat/
├── nginx/
│   ├── Dockerfile          # nginx image + edge routing config
│   └── nginx.conf          # proxy /ws, /api → server; / → web
├── web/
│   ├── Dockerfile          # npm build + nginx for static files
│   ├── nginx.conf          # SPA try_files (internal to web container)
│   └── src/                # React UI
├── server/
│   ├── Dockerfile
│   ├── main.go             # WebSocket hub + API
│   └── db.go               # Postgres connection
├── client/                 # Fyne desktop app
├── migrations/             # SQL migrations
├── docker-compose.yml
├── go.mod, go.sum          # single Go module (server + client)
├── .env.example
└── README.md
```

## HTTP routes (via nginx on :8080)

| Path | Handler | Description |
|------|---------|-------------|
| `GET /` | React (`web`) | Chat test UI in the browser |
| `GET /ws?name=<name>` | Go (`server`) | WebSocket; `name` is **required** |
| `GET /api/health` | Go (`server`) | Health check, `{"ok":true}` |

**WebSocket message format:** plain text. The server broadcasts to other clients as:

```
<sender-name>: <message>
```

The sender does not receive their own message back from the server.

## Requirements

- **Go** 1.21+ (module declares 1.25.6)
- **Docker** and **Docker Compose V2** (v2.22+ for `develop.watch`; v2.23+ for `up --watch`)
- **Node.js 22+** (only if you develop the web UI with Vite on the host)
- **Linux desktop dependencies** for the Fyne client

On Ubuntu/Debian:

```bash
sudo apt install gcc libgl1-mesa-dev xorg-dev
```

On Fedora:

```bash
sudo dnf install gcc mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
```

## Configuration

Copy the example env file and adjust if needed:

```bash
cp .env.example .env
```

The `.env` file is gitignored. Compose loads it via `env_file` and `${POSTGRES_*}` substitution.

| Variable | Role |
|----------|------|
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | PostgreSQL credentials |
| `POSTGRES_HOST` | `db` inside Docker; `localhost` when running Go on the host |
| `POSTGRES_PORT` | Default `5432` |
| `DATABASE_URL` | DSN for the Go server (`pgx`) |

Schema changes are applied by the **migrate** service, not by the server.

## Quick start with Docker

From the project root:

```bash
cp .env.example .env
docker compose up --build
```

Startup order: **db** (healthcheck) → **migrate** (`up`) → **server** → **web** → **nginx**.

- Web UI: http://localhost:8080  
- Desktop client: `ws://localhost:8080/ws` (see `client/main.go`)  
- Server logs should include: `postgres: connected`

Stop containers (keep DB data):

```bash
docker compose down
```

Remove the database volume:

```bash
docker compose down -v
```

## Development: auto-rebuild

`develop.watch` in `docker-compose.yml` rebuilds images when files change. It does **not** run on a plain `docker compose up`.

```bash
docker compose watch
```

or (Compose v2.23+):

```bash
docker compose up --watch
```

| Service | Watched paths |
|---------|----------------|
| `server` | `./server`, `./go.mod`, `./go.sum` |
| `web` | `./web/src`, `./web/package.json` |
| `nginx` | `./nginx/nginx.conf` |

After a rebuild, WebSocket clients must reconnect. Expect several seconds per rebuild.

### Web UI with Vite (faster frontend iteration)

Run the backend in Docker, UI on the host:

```bash
docker compose up db migrate server nginx
cd web && npm install && npm run dev
```

Open http://localhost:5173. Vite proxies `/ws` and `/api` to http://localhost:8080 (nginx).

## Migrations

Migrations run automatically before **server** starts.

Files: `migrations/000001_*.up.sql` / `.down.sql`.

### Manual migrate (Compose)

```bash
set -a && source .env && set +a

docker compose run --rm migrate \
  -path=/migrations \
  -database="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable" \
  up
```

Rollback one version:

```bash
docker compose run --rm migrate \
  -path=/migrations \
  -database="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable" \
  down 1
```

### Local CLI (optional)

Install [golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate). Postgres must be reachable (e.g. temporarily publish `db` ports or use `docker compose exec`).

## Quick start without Docker (Go on host)

**Terminal 1 — database + migrations:**

```bash
cp .env.example .env
# For Go on the host, Postgres must be reachable. Easiest: publish db port in compose
# or run: docker compose up db migrate
export POSTGRES_HOST=localhost
```

**Terminal 2 — server** (no nginx; WebSocket directly on Go):

```bash
set -a && source .env && set +a
export POSTGRES_HOST=localhost
export DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"

go run ./server -addr localhost:8080
```

**Terminal 3 — desktop client:**

```bash
go run ./client
```

For this mode, either point Fyne at `ws://localhost:8080/ws` (Go only) or run the full stack with nginx and keep `ws://localhost:8080/ws` as in `client/main.go`.

## How to verify

### Test A — Two desktop clients

1. Stack is running (`docker compose up` or Go on host with `DATABASE_URL`).
2. Run `go run ./client` twice with different names.
3. Send a message from one client.

**Expected:** The other client shows `Name: message`.

### Test B — Browser + desktop

1. Open http://localhost:8080.
2. Enter a name, **Open**, then **Send**.
3. Desktop client with another name should receive the message.

### Test C — Server logs

```
online: Alice (total: 1)
offline: Alice
```

### Test D — Missing name

```bash
curl -i http://localhost:8080/ws
```

**Expected:** `400 Bad Request` with `name required`.

### Test E — API and database

```bash
curl -s http://localhost:8080/api/health
```

**Expected:** `{"ok":true}`

Inspect tables (Postgres is not published to the host by default):

```bash
docker compose exec db psql -U chat -d chat -c '\dt'
```

Use your `POSTGRES_USER` / `POSTGRES_DB` from `.env` if different.

## Build binaries locally

```bash
go build -o bin/server ./server
go build -o bin/client ./client

export DATABASE_URL=...   # required for the server
./bin/server -addr localhost:8080
./bin/client
```
