package types

import (
	"sync"

	"github.com/gorilla/websocket"
)

// User represents a participant in a room.
type User struct {
	ID             string
	Name           string
	Conn           *websocket.Conn
	MessageChannel chan *Message
}

// MessageType defines allowed message types.
type MessageType string

const (
	MessageTypeChat        MessageType = "chat"
	MessageTypeHostUpdate  MessageType = "host_update"
	MessageTypeSyncRequest MessageType = "sync_req"
	MessageTypeVideoPlay   MessageType = "video_play"
	MessageTypeVideoPause  MessageType = "video_pause"
	MessageTypeVideoSeek   MessageType = "video_seek"
	MessageTypeVideoChange MessageType = "video_change"
)

// Message represents a chat or control message sent via WebSocket.
type Message struct {
	FromUser  *User                  `json:"user"` // Pointer for consistency and efficiency.
	Type      MessageType            `json:"type"` // Use enum for safety.
	Data      map[string]interface{} `json:"data"` // Flexible payload for different message types.
	RoomID    string                 `json:"room_id"`
	Timestamp int64                  `json:"time_stamp"` // Unix timestamp for ordering/history.
}

// Room represents a watch party room.
type Room struct {
	RoomID       string
	Host         *User            // Store host's user ID for quick checks.
	Members      map[string]*User // userID -> *User, for fast lookup and removal.
	VideoURL     string
	VideoState   string       // "playing", "paused", etc.
	VideoRuntime float64      // Current playback position in seconds.
	Messages     []*Message   // Chat and event history (optional, for new joiners).
	Mu           sync.RWMutex // For concurrent access to room state.
}
