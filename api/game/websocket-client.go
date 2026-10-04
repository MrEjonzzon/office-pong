package game

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	ID string `json:"id"`
	// Score     map[string]int    `json:"score"`
	Serving   string            `json:"serving,omitempty"`
	Status    string            `json:"status"` // playing, paused, finished
	Version   int               `json:"version"`
	Players   map[string]Player `json:"players"`
	BestOf    int               `json:"bestOf"`
	Sets      []map[string]int  `json:"sets"` // completed sets, uid -> points
	UpdatedAt time.Time         `json:"updatedAt"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// Envelope wraps all websocket messages
type Envelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// ScoreUpdate uses userId to identify the player to update
type ScoreUpdate struct {
	UserID string `json:"userId"`
	Delta  int    `json:"delta"`
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
		Status:    "playing",
		BestOf:    bestOf,
		Sets:      []map[string]int{},
		Version:   1,
		UpdatedAt: time.Now(),
	}
	return h
}

func (h *Hub) snapshot() GameState {
	h.stateMu.RLock()
	defer h.stateMu.RUnlock()
	return h.state
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

func (h *Hub) applyScoreUpdate(update ScoreUpdate) GameState {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()

	// no scoring until both players are in, or after the match is over
	if h.state.Status == "finished" || len(h.state.Players) < 2 {
		return h.state
	}

	player, ok := h.state.Players[update.UserID]
	if !ok {
		return h.state
	}

	// clamp delta to reasonable bounds
	if update.Delta > 1 {
		update.Delta = 1
	}
	if update.Delta < -1 {
		update.Delta = -1
	}
	player.Score += update.Delta
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
			h.state.Status = "finished"
			h.state.Meta = map[string]string{"winner": player.UID}

			// set scores from the winner's point of view, e.g. "11-7, 9-11, 12-10"
			parts := make([]string, 0, len(h.state.Sets))
			for _, set := range h.state.Sets {
				parts = append(parts, fmt.Sprintf("%d-%d", set[player.UID], set[opponent.UID]))
			}

			go func(db *sql.DB, gameID, winnerID, loserID string, bestOf int, score string) {
				winnerMMR, loserMMR, err := mdb.UpdateMMRAfterGame(db, winnerID, loserID, bestOf)
				if err != nil {
					fmt.Println("[websocket-client.go] error updating MMR", err)
					return
				}
				fmt.Printf("MMR UPDATE - winner %s: %d, loser: %s: %d\n", winnerID, winnerMMR, loserID, loserMMR)
				if err := mdb.CompleteGame(db, gameID, winnerID, score); err != nil {
					fmt.Println("[websocket-client.go] error completing game", err)
				}
			}(h.db, h.ID, player.UID, opponent.UID, h.state.BestOf, strings.Join(parts, ", "))
		}
	}

	h.state.Version++
	h.state.UpdatedAt = time.Now()

	fmt.Println("New game state:", h.state)
	return h.state
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
	bestOf, err := mdb.GetGameBestOf(db, id)
	if err != nil || !ValidBestOf(bestOf) {
		bestOf = 1
	}
	h := newHub(id, db, bestOf)
	hubs[id] = h
	go h.run()
	return h
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
				// If userId not provided, default to sender uid
				if upd.UserID == "" {
					upd.UserID = uid
				}
				newState := h.applyScoreUpdate(upd)
				// broadcast the full new state
				h.broadcast <- pack("state", newState)
			case "state:request":
				// push a broadcast with current state
				h.broadcast <- pack("state", h.snapshot())
			default:
				// ignore unknown types
			}
		}
	}
}
