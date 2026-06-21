package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Local collaboration tool; accept any origin.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client is one connected browser tab.
type Client struct {
	hub   *Hub
	conn  *websocket.Conn
	send  chan []byte
	id    string
	color string
	name  string
	// camera is the last camera state we received, handed to new joiners.
	camera *CameraState
}

// clientMessage is what the browser sends us.
type clientMessage struct {
	Type   string      `json:"type"`
	Camera CameraState `json:"camera"`
	Name   string      `json:"name"`
}

const maxNameLen = 32

func serveWS(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade:", err)
		return
	}
	id, color := hub.assign()
	// Default display name is the id until the user picks one.
	c := &Client{hub: hub, conn: conn, send: make(chan []byte, 16), id: id, color: color, name: id}
	hub.register <- c

	go c.writePump()
	go c.readPump()
	log.Printf("%s connected", id)
}

// readPump reads camera updates from the browser and forwards them to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		log.Printf("%s disconnected", c.id)
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg clientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "camera":
			c.hub.updates <- inbound{client: c, camera: msg.Camera}
		case "name":
			name := strings.TrimSpace(msg.Name)
			if name == "" {
				continue
			}
			if len(name) > maxNameLen {
				name = name[:maxNameLen]
			}
			c.hub.rename <- renameReq{client: c, name: name}
		}
	}
}

// writePump sends queued messages to the browser and keeps the link alive with pings.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
