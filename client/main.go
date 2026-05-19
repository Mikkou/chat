package main

import (
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/gorilla/websocket"
)

const wsBaseURL = "ws://localhost:8080/ws"

type chatClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *chatClient) connect(name string, onMessage func(string)) error {
	wsURL := wsBaseURL + "?name=" + url.QueryEscape(name)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	go func() {
		defer c.close()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fyne.Do(func() {
					onMessage("Connection is closed: " + err.Error())
				})
				return
			}
			text := string(msg)
			fyne.Do(func() {
				onMessage(text)
			})
		}
	}()

	return nil
}

func (c *chatClient) send(text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return errors.New("not connected")
	}
	return c.conn.WriteMessage(websocket.TextMessage, []byte(text))
}

func (c *chatClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func main() {
	a := app.New()
	w := a.NewWindow("Chat")
	w.Resize(fyne.NewSize(600, 500))
	w.SetFixedSize(true)
	w.SetContent(widget.NewLabel("Loading..."))

	var client *chatClient

	var askName func()
	var showChat func(string)

	showChat = func(name string) {
		chatLog := widget.NewMultiLineEntry()
		//chatLog.Disable()
		chatLog.Wrapping = fyne.TextWrapWord

		appendLine := func(line string) {
			if chatLog.Text != "" {
				chatLog.SetText(chatLog.Text + "\n" + line)
			} else {
				chatLog.SetText(line)
			}
		}

		input := widget.NewEntry()
		input.SetPlaceHolder("Message...")

		sendBtn := widget.NewButton("Send", nil)
		send := func() {
			text := strings.TrimSpace(input.Text)
			if text == "" {
				return
			}
			appendLine(name + ": " + text)
			input.SetText("")
			if err := client.send(text); err != nil {
				appendLine("Failed: " + err.Error())
			}
		}
		sendBtn.OnTapped = send
		input.OnSubmitted = func(string) { send() }

		bottom := container.NewBorder(nil, nil, nil, sendBtn, input)
		w.SetContent(container.NewBorder(nil, bottom, nil, nil, chatLog))

		client = &chatClient{}
		go func() {
			if err := client.connect(name, appendLine); err != nil {
				fyne.Do(func() {
					dialog.ShowError(err, w)
					client.close()
					client = nil
					askName()
				})
			}
		}()
	}

	askName = func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Your name")

		dialog.ShowForm("Вход", "Connect", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
		}, func(ok bool) {
			if !ok {
				a.Quit()
				return
			}

			name := strings.TrimSpace(nameEntry.Text)
			if name == "" {
				dialog.ShowInformation("Error", "Write your name", w)
				askName()
				return
			}

			w.SetContent(widget.NewLabel("Conneting..."))
			showChat(name)
		}, w)
	}

	w.SetCloseIntercept(func() {
		if client != nil {
			client.close()
		}
		w.Close()
	})

	go func() {
		time.Sleep(100 * time.Millisecond)
		fyne.Do(askName)
	}()

	w.ShowAndRun()
}
