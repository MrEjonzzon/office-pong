package game

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"strings"
	"sync"
	"time"

	mdb "office-pong/database"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// GameState is the per-game JSON state replicated to all clients
type GameState struct {
	ID         string            `json:"id"`
	Serving    string            `json:"serving,omitempty"`
	Status     string            `json:"status"` // same values as games.state
	Version    int               `json:"version"`
	Players    map[string]Player `json:"players"`
	BestOf     int               `json:"bestOf"`
	Sets       []map[string]int  `json:"sets"` // completed sets, uid -> points
	UpdatedAt  time.Time         `json:"updatedAt"`
	Meta       map[string]string `json:"meta,omitempty"`
	ProposedBy string            `json:"proposedBy,omitempty"`
	Score      string            `json:"score,omitempty"`
}

// Envelope wraps all websocket messages
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type ScoreUpdate struct {
	Delta int `json:"delta"`
}

type Participants struct {
	Count int `json:"count"`
}

type Player struct {
	UID     string `json:"uid"`
	Score   int    `json:"score"` // points in the current set
	SetsWon int    `json:"setsWon"`
	Name    string `json:"name"`
	Image   string `json:"image"`
}

// internal struct to register clients with their uid
type clientJoin struct {
	conn   *websocket.Conn
	client Player
}

// Hub represents a single game's websocket room
type Hub struct {
	ID         string
	db         *sql.DB
	clients    map[*websocket.Conn]string // conn -> uid
	register   chan clientJoin
	unregister chan *websocket.Conn
	broadcast  chan []byte

	state   GameState
	stateMu sync.RWMutex
}

func newHub(id string, db *sql.DB, bestOf int) *Hub {
	h := &Hub{
		ID:         id,
		db:         db,
		clients:    make(map[*websocket.Conn]string),
		register:   make(chan clientJoin),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan []byte, 32),
	}
	// initialize default state
	h.state = GameState{
		ID:        id,
		Players:   make(map[string]Player),
		Serving:   "",
		Status:    "ongoing",
		BestOf:    bestOf,
		Sets:      []map[string]int{},
		Version:   1,
		UpdatedAt: time.Now(),
	}
	return h
}

// loadFromGame copies the stored game state into the hub, so a new or refreshed hub matches the database.
func (h *Hub) loadFromGame(g *mdb.Game) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	h.state.Status = g.State
	h.state.ProposedBy = deref(g.ProposedBy)
	h.state.Score = deref(g.Score)
	h.state.Meta = nil
	if g.WinnerID != nil {
		h.state.Meta = map[string]string{"winner": *g.WinnerID}
	}
	// a rejected result sends the game back to ongoing, and the players replay from 0-0
	if g.State == "ongoing" {
		for uid, p := range h.state.Players {
			p.Score, p.SetsWon = 0, 0
			h.state.Players[uid] = p
		}
		h.state.Sets = []map[string]int{}
	}
	h.state.Version++
	h.state.UpdatedAt = time.Now()
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *Hub) snapshot() GameState {
	h.stateMu.RLock()
	defer h.stateMu.RUnlock()
	s := h.state
	// the copy is sent without the lock, so it must not share the live map
	s.Players = maps.Clone(h.state.Players)
	return s
}

func (h *Hub) ensurePlayer(player Player) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	if h.state.Players == nil {
		h.state.Players = make(map[string]Player)
	}
	// Check if player already exists in state.Players
	if _, ok := h.state.Players[player.UID]; ok {
		return
	}

	h.state.Players[player.UID] = player
	//TODO : singla slant :D
	// if h.state.Serving == "" {
	// 	// First player to join serves first
	// 	h.state.Serving = player.UID
	// }
	h.state.Version++
	h.state.UpdatedAt = time.Now()
}

func (h *Hub) applyScoreUpdate(uid string, delta int) {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	// no scoring until both players are in, or once a result is proposed
	if h.state.Status != "ongoing" || len(h.state.Players) < 2 {
		return
	}

	player, ok := h.state.Players[uid]
	if !ok {
		return
	}

	player.Score += max(-1, min(1, delta))
	if player.Score < 0 {
		player.Score = 0
	}
	h.state.Players[player.UID] = player

	var opponent Player
	for uid, p := range h.state.Players {
		if uid != player.UID {
			opponent = p
		}
	}

	if SetWinner(player.Score, opponent.Score) == 1 {
		h.state.Sets = append(h.state.Sets, map[string]int{
			player.UID:   player.Score,
			opponent.UID: opponent.Score,
		})
		player.SetsWon++
		player.Score = 0
		opponent.Score = 0
		h.state.Players[player.UID] = player
		h.state.Players[opponent.UID] = opponent

		if player.SetsWon >= SetsToWin(h.state.BestOf) {
			sets := make([]string, len(h.state.Sets))
			for i, set := range h.state.Sets {
				sets[i] = fmt.Sprintf("%d-%d", set[player.UID], set[opponent.UID])
			}
			score := strings.Join(sets, ", ")
			if err := mdb.ProposeResult(h.db, h.ID, uid, player.UID, score); err != nil {
				fmt.Println("could not save proposed result for game", h.ID, ":", err)
			} else {
				h.state.Status = "awaiting_confirmation"
				h.state.ProposedBy = uid
				h.state.Score = score
				h.state.Meta = map[string]string{"winner": player.UID}
			}
		}
	}

	h.state.Version++
	h.state.UpdatedAt = time.Now()

	fmt.Println("New game state:", h.state)
}

func raw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return json.RawMessage(b)
}

func pack(t string, payload any) []byte {
	e := Envelope{Type: t}
	if payload != nil {
		e.Payload = raw(payload)
	}
	b, _ := json.Marshal(e)
	return b
}

func (h *Hub) run() {
	for {
		select {
		case join := <-h.register:
			// track client and ensure it exists in the state
			h.clients[join.conn] = join.client.UID
			h.ensurePlayer(join.client)
			// send current state to the newly joined client
			_ = join.conn.WriteMessage(websocket.TextMessage, pack("state", h.snapshot()))

			// notify participant count to everyone
			msg := pack("participants", Participants{Count: len(h.clients)})
			for conn := range h.clients {
				_ = conn.WriteMessage(websocket.TextMessage, msg)
			}
			// also broadcast updated state so others see the new player
			stateMsg := pack("state", h.snapshot())
			for conn := range h.clients {
				_ = conn.WriteMessage(websocket.TextMessage, stateMsg)
			}
		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				_ = c.Close()
			}
			// notify participant count after leave
			msg := pack("participants", Participants{Count: len(h.clients)})
			for conn := range h.clients {
				_ = conn.WriteMessage(websocket.TextMessage, msg)
			}

		case msg := <-h.broadcast:
			for conn := range h.clients {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					// If write fails, drop the connection
					delete(h.clients, conn)
					_ = conn.Close()
				}
			}
		}
	}
}

// Global hub registry
var (
	hubs   = map[string]*Hub{}
	hubsMu sync.RWMutex
)

// CreateHub ensures a hub exists for a game id and returns it.
func CreateHub(id string, db *sql.DB) *Hub {
	hubsMu.Lock()
	defer hubsMu.Unlock()
	if h, ok := hubs[id]; ok {
		return h
	}
	g, err := mdb.GetGame(db, id)
	bestOf := 1
	if err == nil && ValidBestOf(g.BestOf) {
		bestOf = g.BestOf
	}
	h := newHub(id, db, bestOf)
	if err == nil {
		h.loadFromGame(g)
	}
	hubs[id] = h
	go h.run()
	return h
}

// RefreshHub reloads an open hub from the games row and sends the new state to its clients.
func RefreshHub(db *sql.DB, gameID string) {
	h, ok := GetHub(gameID)
	if !ok {
		return
	}
	g, err := mdb.GetGame(db, gameID)
	if err != nil {
		fmt.Println("could not reload game", gameID, ":", err)
		return
	}
	h.loadFromGame(g)
	h.broadcast <- pack("state", h.snapshot())
}

// GetHub retrieves an existing hub by id.
func GetHub(id string) (*Hub, bool) {
	hubsMu.RLock()
	defer hubsMu.RUnlock()
	h, ok := hubs[id]
	return h, ok
}

func closeConnection(conn *websocket.Conn, message string) {
	fmt.Println("Closing connection with message:", message)
	// Send close message with status code 1008 (Policy Violation)
	err := conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, message),
		time.Now().Add(time.Second))
	if err != nil {
		fmt.Println("Error sending close message:", err)
	}
	// Explicitly close the connection
	_ = conn.Close()
	fmt.Println("Connection closed")
}

// WSHandler attaches a client to the hub for the given game id
func WSHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("initiating websocket connection")
		vars := mux.Vars(r)
		gameID := vars["id"]
		fmt.Println("game ID :", gameID)
		if gameID == "" {
			http.Error(w, "missing game id", http.StatusBadRequest)
			return
		}

		// Upgrade the HTTP connection to a WebSocket connection
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Error upgrading:", err)
			return
		}
		defer conn.Close()

		authToken := r.URL.Query().Get("token")
		if authToken == "" || authToken == "undefined" {
			closeConnection(conn, "no token provided")
			return
		}

		// Verify auth token
		uid, err := mdb.VerifyUserToken(db, authToken)
		if err != nil || uid == "" {
			closeConnection(conn, "invalid token")
			return
		}

		// Grab user details
		user, err := mdb.GetUserByID(db, uid)
		if err != nil {
			closeConnection(conn, "failed to get user details")
			return
		}

		fmt.Println("Authenticated user ID:", uid, "joining game:", gameID)

		h := CreateHub(gameID, db)
		h.register <- clientJoin{conn: conn, client: Player{UID: uid, Name: user.Name, Image: user.Image}}
		defer func() { h.unregister <- conn }()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("Error reading message:", err)
				return
			}

			var env Envelope
			if err := json.Unmarshal(message, &env); err != nil {
				// ignore malformed
				fmt.Println("Malformed message:", err)
				continue
			}
			fmt.Println("env received:", env)

			switch env.Type {
			case "score:update":
				fmt.Println("Received score update:", string(env.Payload))
				var upd ScoreUpdate
				if err := json.Unmarshal(env.Payload, &upd); err != nil {
					continue
				}
				h.applyScoreUpdate(uid, upd.Delta)
				h.broadcast <- pack("state", h.snapshot())
			case "state:request":
				// push a broadcast with current state
				h.broadcast <- pack("state", h.snapshot())
			default:
				// ignore unknown types
			}
		}
	}
}
