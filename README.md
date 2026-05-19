# WebSocket Chat

A small real-time chat demo: a Go WebSocket server that broadcasts messages between clients, a Fyne desktop app, and a built-in browser test page. The server connects to PostgreSQL on startup (ping only; it does not read or write chat data).

## Project structure

```
chat/
├── server/              # WebSocket hub + HTTP test UI + DB connection
├── client/              # Fyne desktop chat app
├── migrations/          # SQL migrations (golang-migrate)
├── docker-compose.yml
├── Dockerfile
├── .env.example         # copy to .env (not committed)
├── go.mod
└── README.md
```

## Requirements

- **Go** 1.21 or newer (module declares 1.25.6)
- **Docker** and **Docker Compose V2** (v2.22+ for `develop.watch`; v2.23+ for `up --watch`)
- **Linux desktop dependencies** for the Fyne client (compiler + OpenGL + X11/Wayland dev packages)

On Ubuntu/Debian:

```bash
sudo apt install gcc libgl1-mesa-dev xorg-dev
```

On Fedora:

```bash
sudo dnf install gcc mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
```

## Configuration

All credentials live in `.env` (see `.env.example`). Docker Compose reads them via `env_file` and `${POSTGRES_*}` substitution. The `.env` file is gitignored.

```bash
cp .env.example .env
```

| Variable | Role |
|----------|------|
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | PostgreSQL credentials |
| `POSTGRES_HOST` | `db` inside Docker; `localhost` when running the server on the host |
| `POSTGRES_PORT` | Default `5432` |
| `DATABASE_URL` | DSN for the Go server (`pgx`) |

The application only opens a connection and pings PostgreSQL at startup. Schema changes are applied by the **migrate** service, not by the server.

## Quick start with Docker

From the project root:

```bash
cp .env.example .env
docker compose up --build
```

Startup order: **db** (healthcheck) → **migrate** (`up`) → **server** on http://localhost:8080.

You should see `postgres: connected` (or similar) in the server logs. WebSocket and the test UI work the same as without Docker.

Stop and remove containers (keep the DB volume):

```bash
docker compose down
```

Remove the database volume as well:

```bash
docker compose down -v
```

## Development: auto-rebuild on server changes

`docker-compose.yml` defines `develop.watch` for the **server** service. It only works when watch mode is enabled — a plain `docker compose up` does **not** reload code.

```bash
docker compose watch
```

or (Compose v2.23+):

```bash
docker compose up --watch
```

Watched paths: `./server`, `./go.mod`. On change, Compose rebuilds the image and recreates the **server** container. Expect a few seconds per rebuild; WebSocket clients must reconnect after a restart.

## Migrations

Migrations run automatically via the **migrate** container (`migrate/migrate` image) before the server starts.

Files live in `migrations/` (`000001_*.up.sql` / `.down.sql`).

### Run migrations manually

With the stack running:

```bash
docker compose run --rm migrate \
  -path=/migrations \
  -database="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable" \
  up
```

Rollback one step:

```bash
docker compose run --rm migrate \
  -path=/migrations \
  -database="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable" \
  down 1
```

### Local CLI (optional)

Install [golang-migrate](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate), set `DATABASE_URL` (with `POSTGRES_HOST=localhost` if Postgres is exposed on the host), then:

```bash
migrate -path migrations -database "$DATABASE_URL" up
```

## Quick start without Docker (server on host)

You still need PostgreSQL and `DATABASE_URL`. Easiest: run only the database in Docker, server locally.

**Terminal 1 — database + migrations:**

```bash
cp .env.example .env
# edit .env: POSTGRES_HOST=localhost for local server
docker compose up db migrate
```

**Terminal 2 — server:**

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

1. A window opens and a **login dialog** asks for your name.
2. Click **Connect**.
3. The main chat screen appears after a successful WebSocket connection.

If you change the server port, update `wsBaseURL` in `client/main.go`.

## How to verify everything works

### Test A — Two desktop clients

1. Server is running (Docker or `go run ./server` with `DATABASE_URL` set).
2. Start the client twice.
3. Enter different names (e.g. `Alice` and `Bob`).
4. Send a message from Alice.

**Expected:** Bob’s window shows `Alice: <message>`. Alice sees her own line locally but does not receive it back from the server (broadcast skips the sender).

### Test B — Desktop + browser

1. Server is running.
2. Open http://localhost:8080 in a browser.
3. Enter a name, click **Open**, then **Send**.
4. Desktop client (another name) should receive `BrowserName: <message>`.

### Test C — Server logs

On connect:

```
online: Alice (total: 1)
```

On disconnect:

```
offline: Alice
```

### Test D — Missing name (negative test)

```bash
curl -i http://localhost:8080/echo
```

**Expected:** HTTP `400 Bad Request` with `name required`.

### Test E — PostgreSQL

After `docker compose up`, server logs should show a successful DB connection. The server does not insert chat messages; to inspect the schema:

```bash
docker compose exec db psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c '\dt'
```

(Load variables from `.env` or substitute your user/db names.)

## WebSocket API

| Endpoint | Description |
|----------|-------------|
| `GET /` | HTML page for manual WebSocket testing |
| `GET /echo?name=<name>` | WebSocket upgrade; `name` query parameter is **required** |

**Message format:** plain text body. The server broadcasts to all other clients as:

```
<sender-name>: <message>
```

## Run without `go run`

```bash
go build -o bin/server ./server
go build -o bin/client ./client

export DATABASE_URL=...   # required for the server
./bin/server -addr localhost:8080
./bin/client
```