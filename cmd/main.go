package main

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/theweird-kid/teleparty/handlers"
	"github.com/theweird-kid/teleparty/services"
)

func main() {
	fmt.Println("teleparty")
	r := gin.New()
	r.Use(cors.Default())

	app := &handlers.Application{
		RoomManager: services.NewRoomManager(),
	}

	r.GET("/join/:room_id", app.JoinRoom)
	r.GET("/create", app.CreateRoom)

	r.Run("127.0.0.1:8080")
}
