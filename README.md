# WebSocket Chat

A small real-time chat demo: a Go WebSocket server that broadcasts messages between clients, a Fyne desktop app, and a built-in browser test page.

## Project structure

```
test-golang/
├── server/     # WebSocket hub + HTTP test UI
├── client/     # Fyne desktop chat app
├── go.mod
└── README.md
```

## Requirements

- **Go** 1.21 or newer (module declares 1.25.6)
- **Linux desktop dependencies** for the Fyne client (compiler + OpenGL + X11/Wayland dev packages)

On Ubuntu/Debian:

```bash
sudo apt install gcc libgl1-mesa-dev xorg-dev
```

On Fedora:

```bash
sudo dnf install gcc mesa-libGL-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libXxf86vm-devel
```

## Quick start

Open **two terminals** in the project root (`chat/`).

### 1. Start the server

```bash
go run ./server
```

The server listens on `localhost:8080` by default. You should see no errors; connections are logged when clients join or leave.

Custom address:

```bash
go run ./server -addr localhost:9090
```

If you change the port, update `wsBaseURL` in `client/main.go` to match.

### 2. Start the desktop client

In the second terminal:

```bash
go run ./client
```

1. A window opens and a **login dialog** asks for your name.
2. Click **Connect** (or **OK** if using the entry dialog).
3. The main chat screen appears after a successful WebSocket connection.

## How to verify everything works

### Test A — Two desktop clients

1. Start the server (`go run ./server`).
2. Start the client twice (two terminals, or run the binary twice).
3. Enter different names (e.g. `Alice` and `Bob`).
4. Send a message from Alice.

**Expected:** Bob’s window shows `Alice: <message>`. Alice sees her own line locally as `Alice: <message>` but does not receive it back from the server (broadcast skips the sender).

### Test B — Desktop + browser

1. Server is running.
2. Open http://localhost:8080 in a browser.
3. Enter a name, click **Open**, then **Send**.
4. Desktop client (with another name) should receive `BrowserName: <message>`.

**Expected:** Messages typed in the browser appear in the Fyne app, and messages from the desktop app appear in the browser output panel.

### Test C — Server logs

When a client connects, the server prints something like:

```
online: Alice (total: 1)
```

When they disconnect:

```
offline: Alice
```

**Expected:** `total` increases with each new connection and decreases when clients close.

### Test D — Missing name (negative test)

Connect without `?name=` (e.g. `curl -i http://localhost:8080/echo`).

**Expected:** HTTP `400 Bad Request` with `name required`.

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

./bin/server
./bin/client
```
