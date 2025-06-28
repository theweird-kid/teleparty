package types

import (
	"github.com/gorilla/websocket"
)

// User represents a participant in a room.
type User struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Conn           *websocket.Conn `json:"-"`
	MessageChannel chan *Message   `json:"-"`
}

// MessageType defines allowed message types.
type MessageType string

const (
	MessageTypeChat           MessageType = "chat_message_broadcast"
	MessageTypeHostUpdate     MessageType = "host_change_broadcast"
	MessageTypeSyncRequest    MessageType = "sync_request"
	MessageTypeSyncResponse   MessageType = "sync_response"
	MessageTypeVideoURLChange MessageType = "video_url_change_broadcast"
	MessageTypeVideoPlayPause MessageType = "video_play_pause_broadcast"
	MessageTypeVideoSeek      MessageType = "video_seek_broadcast"
)

// Message represents a chat or control message sent via WebSocket.
type Message struct {
	FromUser  *User                  `json:"user"` // Pointer for consistency and efficiency.
	Type      MessageType            `json:"type"` // Use enum for safety.
	Data      map[string]interface{} `json:"data"` // Flexible payload for different message types.
	RoomID    string                 `json:"room_id"`
	Timestamp int64                  `json:"time_stamp"` // Unix timestamp for ordering/history.
}

type RoomCommandType int

const (
	CmdJoin RoomCommandType = iota
	CmdLeave
	CmdSyncReq
	CmdBroadcast
	CmdHostChange
	// Add more as needed
)

type RoomCommand struct {
	Type    RoomCommandType
	User    *User
	Message *Message
	Reply   chan interface{} // Optional: for synchronous commands
}

// Room represents a watch party room.
type Room struct {
	RoomID       string
	Host         *User            // Store host's user ID for quick checks.
	Members      map[string]*User // userID -> *User, for fast lookup and removal.
	VideoURL     string
	IsPlaying    bool       // "playing", "paused", etc.
	VideoRuntime float64    // Current playback position in seconds.
	Messages     []*Message // Chat and event history (optional, for new joiners).
}
