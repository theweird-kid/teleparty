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
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// allow only your frontend's domain
		switch origin {
		case "https://teleparty-app.onrender.com":
			return true
		default:
			return false
		}
	},
}

// join room websocket connection
func (app *Application) JoinRoom(c *gin.Context) {
	roomID := c.Param("room_id")
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	entry, ok := app.RoomManager.GetRoomEntry(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	entry.Room.Mu.RLock()
	user, exists := entry.Room.Members[userID]
	entry.Room.Mu.RUnlock()
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found in room"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "websocket upgrade failed"})
		return
	}
	user.Conn = conn

	// Start goroutine to handle this user
	go app.handleUserConnection(user, roomID)
}

// request to join a room (HTTP → registers user → returns IDs)
func (app *Application) RequestJoin(c *gin.Context) {
	name := c.Query("name")
	roomID := c.Param("room_id")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	entry, ok := app.RoomManager.GetRoomEntry(roomID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	user := &types.User{
		ID:             utils.GenerateRandomID(),
		Name:           name,
		MessageChannel: make(chan *types.Message, BUFFER_SIZE),
	}

	// register user
	entry.CmdChan <- types.RoomCommand{
		Type: types.CmdJoin,
		User: user,
	}

	c.JSON(http.StatusOK, gin.H{
		"room_id":   roomID,
		"user_id":   user.ID,
		"user_name": user.Name,
	})
}

// create a new room and host
func (app *Application) CreateRoom(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	host := &types.User{
		ID:             utils.GenerateRandomID(),
		Name:           name,
		MessageChannel: make(chan *types.Message, BUFFER_SIZE),
	}

	room := app.RoomManager.CreateNewRoom(host)

	c.JSON(http.StatusOK, gin.H{
		"room_id":   room.RoomID,
		"user_id":   host.ID,
		"user_name": host.Name,
	})
}

// handles a user websocket connection lifecycle
func (app *Application) handleUserConnection(user *types.User, roomID string) {
	entry, ok := app.RoomManager.GetRoomEntry(roomID)
	if !ok {
		log.Printf("Room %s no longer exists", roomID)
		return
	}

	defer func() {
		entry.CmdChan <- types.RoomCommand{Type: types.CmdLeave, User: user}
		log.Printf("User %s disconnected from room %s\n", user.Name, roomID)
	}()

	// send messages to client
	go func() {
		for msg := range user.MessageChannel {
			if err := user.Conn.WriteJSON(msg); err != nil {
				log.Printf("Error sending message to %s: %v\n", user.Name, err)
				break
			}
		}
	}()

	time.Sleep(2 * time.Second) // optional: allow client to catch up

	// read messages from client
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
		entry.CmdChan <- cmd
	}
}
