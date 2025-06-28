package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/theweird-kid/teleparty/services"
	"github.com/theweird-kid/teleparty/types"
	"github.com/theweird-kid/teleparty/utils"
)

type Application struct {
	RoomManager *services.RoomManager
}

const BUFFER_SIZE int = 16

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // For dev only; restrict in prod!
}

func (app *Application) JoinRoom(c *gin.Context) {
	roomID := c.Param("room_id")
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	room := app.RoomManager.Rooms[roomID]
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	// Get the registered user and write conn
	user := room.Members[userID]
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found in room"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "websocket upgrade failed"})
		return
	}
	user.Conn = conn

	// Start goroutine to handle incoming/outgoing messages for this user
	go app.handleUserConnection(user, roomID)
}

func (app *Application) RequestJoin(c *gin.Context) {
	name := c.Query("name")
	roomID := c.Param("room_id")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	user := &types.User{
		ID:             utils.GenerateRandomID(),
		Name:           name,
		MessageChannel: make(chan *types.Message, 16),
	}
	// Register the user in the room
	cmdChan := app.RoomManager.CmdChans[roomID]
	cmdChan <- types.RoomCommand{
		Type: types.CmdJoin,
		User: user,
	}

	c.JSON(http.StatusOK, gin.H{
		"room_id":   roomID,
		"user_id":   user.ID,
		"user_name": user.Name,
	})
}

func (app *Application) CreateRoom(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	// Create User (host)
	host := &types.User{
		ID:             utils.GenerateRandomID(),
		Name:           name,
		MessageChannel: make(chan *types.Message, 16),
	}
	// Create Room
	roomID := utils.GenerateRandomID()
	room := &types.Room{
		RoomID:       roomID,
		Host:         host,
		Members:      map[string]*types.User{host.ID: host},
		VideoURL:     "",
		VideoRuntime: 0,
		IsPlaying:    false,
	}
	cmdChan := make(chan types.RoomCommand, 32)

	// Register in RoomManager
	app.RoomManager.Mu.Lock()
	app.RoomManager.Rooms[roomID] = room
	app.RoomManager.CmdChans[roomID] = cmdChan
	app.RoomManager.Mu.Unlock()

	go services.RunRoom(room, cmdChan, app.RoomManager.CleanupCh)

	// Optionally: send CmdJoin for host (not strictly needed if you already added above)
	// cmdChan <- types.RoomCommand{Type: types.CmdJoin, User: host}

	c.JSON(http.StatusOK, gin.H{
		"room_id":   roomID,
		"user_id":   host.ID,
		"user_name": host.Name,
	})
}

func (app *Application) handleUserConnection(user *types.User, roomID string) {
	cmdChan := app.RoomManager.CmdChans[roomID]
	defer func() {
		cmdChan <- types.RoomCommand{Type: types.CmdLeave, User: user}
		log.Printf("User %s disconnected from room %s\n", user.Name, roomID)
	}()

	// send replies/Broadcast to clients
	go func() {
		for msg := range user.MessageChannel {
			if err := user.Conn.WriteJSON(msg); err != nil {
				log.Printf("Error sending message to %s: %v\n", user.Name, err)
				break
			}
		}
	}()

	time.Sleep(2 * time.Second)
	// Read messages from client
	for {
		var msg types.Message
		err := user.Conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading message from %s: %v\n", user.Name, err)
			break
		}
		msg.FromUser = user
		msg.RoomID = roomID

		cmd := types.RoomCommand{
			User:    user,
			Message: &msg,
		}
		switch msg.Type {
		case types.MessageTypeSyncRequest:
			cmd.Type = types.CmdSyncReq
		default:
			cmd.Type = types.CmdBroadcast
		}
		cmdChan <- cmd
	}
}
