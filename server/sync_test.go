package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startTestServer spins up the hub behind an httptest server and returns the ws URL.
func startTestServer(t *testing.T) string {
	t.Helper()
	hub := newHub()
	go hub.run()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWS(hub, w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
}

func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// readMsg reads one JSON message, failing if none arrives in time.
func readMsg(t *testing.T, c *websocket.Conn) map[string]any {
	t.Helper()
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %q: %v", data, err)
	}
	return m
}

func TestSyncFlow(t *testing.T) {
	url := startTestServer(t)

	// Client A connects and gets an init with no peers yet.
	a := dial(t, url)
	aInit := readMsg(t, a)
	if aInit["type"] != "init" {
		t.Fatalf("A first message type = %v, want init", aInit["type"])
	}
	aID, _ := aInit["id"].(string)
	if aID == "" {
		t.Fatal("A got no id")
	}
	if peers := aInit["peers"].([]any); len(peers) != 0 {
		t.Fatalf("A should see no peers, got %d", len(peers))
	}

	// Client B connects; its init should list A as an existing peer.
	b := dial(t, url)
	bInit := readMsg(t, b)
	peers := bInit["peers"].([]any)
	if len(peers) != 1 {
		t.Fatalf("B should see 1 existing peer, got %d", len(peers))
	}
	if got := peers[0].(map[string]any)["id"]; got != aID {
		t.Fatalf("B's existing peer id = %v, want %v", got, aID)
	}

	// A moves its camera; B must receive the update for A.
	a.WriteJSON(map[string]any{
		"type": "camera",
		"camera": map[string]any{
			"position":  map[string]float64{"x": 1, "y": 2, "z": 3},
			"direction": map[string]float64{"x": 0, "y": 0, "z": -1},
		},
	})
	peerMsg := readMsg(t, b)
	if peerMsg["type"] != "peer" || peerMsg["id"] != aID {
		t.Fatalf("B got %v for %v, want peer for A", peerMsg["type"], peerMsg["id"])
	}
	cam := peerMsg["camera"].(map[string]any)["position"].(map[string]any)
	if cam["x"].(float64) != 1 || cam["z"].(float64) != 3 {
		t.Fatalf("B received wrong camera position: %v", cam)
	}

	// A disconnects; B must be told to drop A.
	a.Close()
	leaveMsg := readMsg(t, b)
	if leaveMsg["type"] != "leave" || leaveMsg["id"] != aID {
		t.Fatalf("B got %v/%v, want leave for A", leaveMsg["type"], leaveMsg["id"])
	}
}
