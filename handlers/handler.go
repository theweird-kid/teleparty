package handlers

import (
	"fmt"
	"log"
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
		MessageChannel: make(chan *types.Message, 16),
	}
	room := app.RoomManager.CreateNewRoom(host)
	c.JSON(http.StatusOK, gin.H{
		"room_id": room.RoomID,
		"user_id": host.ID,
		"name":    host.Name,
	})
}

func (app *Application) handleUserConnection(user *types.User, room *types.Room) {
	defer func() {
		user.Conn.Close()
		close(user.MessageChannel)
		app.RoomManager.ExitRoom(user, room.RoomID)
		log.Printf("User %s disconnected from room %s\n", user.Name, room.RoomID)
	}()

	// Pickup the message from the user channel and send to the client
	go func() {
		for msg := range user.MessageChannel {
			if err := user.Conn.WriteJSON(msg.Data); err != nil {
				log.Printf("Error sending message to %s: %v\n", user.Name, err)
				break
			}
		}
	}()

	// Read the message from client and broadcast to other user
	for {
		var msg types.Message
		err := user.Conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading message from %s: %v\n", user.Name, err)
			break
		}
		if err := validateMessage(&msg); err != nil {
			log.Printf("Validation error from %s: %v\n", user.Name, err)
			continue // Skip invalid message
		}
		msg.FromUser = user
		msg.RoomID = room.RoomID
		log.Println(msg)
		app.RoomManager.BroadcastToRoom(&msg)
	}
}

func validateMessage(msg *types.Message) error {
	switch msg.Type {
	case types.MessageTypeChat:
		text, ok := msg.Data["text"].(string)
		if !ok || text == "" {
			return fmt.Errorf("invalid chat message: missing or empty text")
		}
	case types.MessageTypeVideoPlay, types.MessageTypeVideoPause:
		// No extra data needed, but you could check for timestamp if required
	case types.MessageTypeVideoSeek:
		_, ok := msg.Data["position"].(float64)
		if !ok {
			return fmt.Errorf("invalid seek message: missing or invalid position")
		}
	// Add more cases as needed
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
	return nil
}
