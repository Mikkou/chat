package main

import (
	"flag"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub — список всех активных подключений.
type Hub struct {
	mu      sync.Mutex
	clients []*Client
}

func (h *Hub) add(c *Client) {
	h.mu.Lock()
	h.clients = append(h.clients, c)
	h.mu.Unlock()
}

func (h *Hub) remove(c *Client) {
	h.mu.Lock()
	for i, x := range h.clients {
		if x == c {
			h.clients = append(h.clients[:i], h.clients[i+1:]...)
			break
		}
	}
	h.mu.Unlock()
}

func (h *Hub) broadcast(from *Client, text string) {
	line := from.Name + ": " + text

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, c := range h.clients {
		if c == from {
			continue
		}
		_ = c.Conn.WriteMessage(websocket.TextMessage, []byte(line))
	}
}

// Client — одно WS-подключение + имя из ?name=
type Client struct {
	Name string
	Conn *websocket.Conn
	hub  *Hub
}

func (c *Client) readLoop() {
	defer func() {
		c.hub.remove(c)
		c.Conn.Close()
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		c.hub.broadcast(c, string(msg))
	}
}

var hub = &Hub{}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	client := &Client{
		Name: name,
		Conn: conn,
		hub:  hub,
	}
	hub.add(client)
	log.Printf("online: %s (total: %d)", name, len(hub.clients))

	client.readLoop()
	log.Printf("offline: %s", name)
}

func apiHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("postgres: connected")

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/api/health", apiHealth)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
