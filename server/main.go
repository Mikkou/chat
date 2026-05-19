package main

import (
	"flag"
	"html/template"
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

func echo(w http.ResponseWriter, r *http.Request) {
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

func home(w http.ResponseWriter, r *http.Request) {
	homeTemplate.Execute(w, "ws://"+r.Host+"/echo")
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

	http.HandleFunc("/echo", echo)
	http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {

    let output = document.getElementById("output");
    let input = document.getElementById("input");
    let name = document.getElementById("name");
    let ws;

    let print = function(message) {
        let d = document.createElement("div");
        d.textContent = message;
        output.appendChild(d);
        output.scroll(0, output.scrollHeight);
    };

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }

		if (!name.value) {
			alert('name is required');
			return false
		}
        ws = new WebSocket("{{.}}?name="+name.value);;
        ws.onopen = function(evt) {
            print("OPEN");
        }
        ws.onclose = function(evt) {
            print("CLOSE");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print(evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };

    document.getElementById("send").onclick = function(evt) {
        if (!ws) {
            return false;
        }

        print(name.value + ": " + input.value);
        ws.send(input.value);
        return false;
    };

    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };

});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Open" to create a connection to the server, 
"Send" to send a message to the server and "Close" to close the connection. 
You can change the message and send multiple times.
<p>
<form>
<button id="open">Open</button>
<button id="close">Close</button>
<input id="name" type="text" placeholder="your name"></input>
<p><input id="input" type="text" value="Hello world!">
<button id="send">Send</button>
</form>
</td><td valign="top" width="50%">
<div id="output" style="max-height: 70vh;overflow-y: scroll;"></div>
</td></tr></table>
</body>
</html>
`))
