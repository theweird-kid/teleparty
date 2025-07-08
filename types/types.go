package types

import (
	"sync"

	"github.com/gorilla/websocket"
)

// User represents a participant in a room.
type User struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Conn           *websocket.Conn `json:"-"` // Do not expose in JSON.
	MessageChannel chan *Message   `json:"-"` // Internal only.
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
	FromUser  *User                  `json:"user"`       // Sender of the message.
	Type      MessageType            `json:"type"`       // Type of message.
	Data      map[string]interface{} `json:"data"`       // Payload.
	RoomID    string                 `json:"room_id"`    // Which room this message belongs to.
	Timestamp int64                  `json:"time_stamp"` // Unix timestamp.
}

// RoomCommandType defines internal room commands.
type RoomCommandType int

const (
	CmdJoin RoomCommandType = iota
	CmdLeave
	CmdSyncReq
	CmdBroadcast
	CmdHostChange
	// Add more as needed.
)

// RoomCommand represents a command sent to the room goroutine.
type RoomCommand struct {
	Type    RoomCommandType
	User    *User
	Message *Message
	Reply   chan interface{} // Optional: for synchronous commands.
}

// Room represents a watch party room.
type Room struct {
	RoomID       string           // Unique room ID.
	Host         *User            // Current host.
	Members      map[string]*User // userID → *User.
	VideoURL     string           // Current video URL.
	IsPlaying    bool             // Playing or paused.
	VideoRuntime float64          // Current playback position in seconds.
	Messages     []*Message       // (optional) chat and event history.
	Mu           sync.RWMutex     // Protects Members and Messages.
}
