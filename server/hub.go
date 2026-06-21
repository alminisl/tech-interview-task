package main

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

// Vec3 is a 3D vector in the point cloud's world coordinate frame. Every client
// loads the same cloud the same way, so raw world coordinates are directly
// comparable between clients without any per-client offset.
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// CameraState is where a peer is and which way it is looking.
type CameraState struct {
	Position  Vec3 `json:"position"`
	Direction Vec3 `json:"direction"`
}

// PeerState is the public view of one client, sent to others.
type PeerState struct {
	ID     string       `json:"id"`
	Color  string       `json:"color"`
	Name   string       `json:"name"`
	Camera *CameraState `json:"camera,omitempty"`
}

// inbound is a camera update arriving from a client.
type inbound struct {
	client *Client
	camera CameraState
}

// renameReq is a display-name change arriving from a client.
type renameReq struct {
	client *Client
	name   string
}

// Hub keeps the set of connected clients and fans camera state out between them.
// All state lives in one goroutine (run), so no locks are needed on the maps.
type Hub struct {
	// nextID is accessed atomically; it must stay the first field so it is
	// 64-bit aligned on 32-bit platforms (Go only guarantees alignment for the
	// first word of a struct).
	nextID int64

	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	updates    chan inbound
	rename     chan renameReq

	palette []string
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		updates:    make(chan inbound),
		rename:     make(chan renameReq),
		// Distinct, well-spaced hues so each peer gets a stable, readable color.
		palette: []string{
			"#e6194b", "#3cb44b", "#4363d8", "#f58231", "#911eb4",
			"#42d4f4", "#f032e6", "#bfef45", "#fabed4", "#469990",
		},
	}
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c] = true
			// Tell the newcomer who it is and where everyone already is.
			c.send <- mustJSON(map[string]any{
				"type":  "init",
				"id":    c.id,
				"color": c.color,
				"peers": h.peerList(c),
			})

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
				// Tell everyone else to drop this peer's marker.
				h.broadcast(c, mustJSON(map[string]any{
					"type": "leave",
					"id":   c.id,
				}))
			}

		case in := <-h.updates:
			in.client.camera = &in.camera
			// Relay this peer's new camera to everyone else.
			h.broadcast(in.client, mustJSON(map[string]any{
				"type":   "peer",
				"id":     in.client.id,
				"color":  in.client.color,
				"name":   in.client.name,
				"camera": in.camera,
			}))

		case rn := <-h.rename:
			rn.client.name = rn.name
			// Tell everyone else this peer's new display name.
			h.broadcast(rn.client, mustJSON(map[string]any{
				"type": "name",
				"id":   rn.client.id,
				"name": rn.name,
			}))
		}
	}
}

// peerList returns the current state of every client except the given one.
func (h *Hub) peerList(except *Client) []PeerState {
	peers := make([]PeerState, 0, len(h.clients))
	for c := range h.clients {
		if c == except {
			continue
		}
		peers = append(peers, PeerState{ID: c.id, Color: c.color, Name: c.name, Camera: c.camera})
	}
	return peers
}

// broadcast sends a message to every client except the sender. A client whose
// send buffer is full is dropped, which unregisters it.
func (h *Hub) broadcast(sender *Client, msg []byte) {
	for c := range h.clients {
		if c == sender {
			continue
		}
		select {
		case c.send <- msg:
		default:
			delete(h.clients, c)
			close(c.send)
		}
	}
}

func (h *Hub) assign() (id, color string) {
	n := atomic.AddInt64(&h.nextID, 1)
	return fmt.Sprintf("peer-%d", n), h.palette[(n-1)%int64(len(h.palette))]
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
