package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/theweird-kid/teleparty/handlers"
	"github.com/theweird-kid/teleparty/services"
)

func main() {
	log.Println("Starting Teleparty")
	allowedOrigins := []string{"http://localhost:5173/*"}
	allowedOrigins = append(allowedOrigins, os.Getenv("ALLOWED_ORIGINS"))
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	rm := services.NewRoomManager()
	rm.StartCleanupListener()
	app := &handlers.Application{
		RoomManager: rm,
	}

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})
	r.GET("/join/:room_id", app.JoinRoom)
	r.GET("/join_req/:room_id", app.RequestJoin)
	r.POST("/create", app.CreateRoom)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{})
	})

	log.Println("Starting server on port 8080...")
	r.Run(":8080")
}
