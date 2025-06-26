package handlers

import (
	"net/http"

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
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	user := &types.User{
		ID:             utils.GenerateRandomID(),
		Name:           name,
		MessageChannel: make(chan *types.Message, BUFFER_SIZE),
	}

	room := app.RoomManager.Rooms[roomID]
	if room == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}
	// Register the user in the room
	app.RoomManager.JoinRoom(user, roomID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "websocket upgrade failed"})
		return
	}
	user.Conn = conn

	// Start goroutine to handle incoming/outgoing messages for this user
	go app.handleUserConnection(user, room)
}

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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "websocket upgrade failed"})
		return
	}
	host.Conn = conn

	go app.handleUserConnection(host, room)
}

func (app *Application) handleUserConnection(user *types.User, room *types.Room) {
	defer func() {
		user.Conn.Close()
		close(user.MessageChannel)
		app.RoomManager.ExitRoom(user, room.RoomID)
	}()

	// Pickup the message from the user channel and send to the client
	go func() {
		for msg := range user.MessageChannel {
			if err := user.Conn.WriteJSON(msg); err != nil {
				break
			}
		}
	}()

	// Read the message from client and broadcast to other user
	for {
		var msg types.Message
		err := user.Conn.ReadJSON(&msg)
		if err != nil {
			break
		}
		msg.FromUser = user
		msg.RoomID = room.RoomID
		app.RoomManager.BroadcastToRoom(&msg)
	}
}
