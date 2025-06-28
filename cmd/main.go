package main

import (
	"fmt"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/theweird-kid/teleparty/handlers"
	"github.com/theweird-kid/teleparty/services"
)

func main() {
	fmt.Println("teleparty")
	r := gin.New()
	r.Use(cors.Default())

	rm := services.NewRoomManager()
	rm.StartCleanupListener()
	app := &handlers.Application{
		RoomManager: rm,
	}

	r.GET("/join/:room_id", app.JoinRoom)
	r.GET("/join_req/:room_id", app.RequestJoin)
	r.POST("/create", app.CreateRoom)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	r.Run("127.0.0.1:8080")
}
