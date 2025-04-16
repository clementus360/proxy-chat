package main

import (
	"log"
	"net/http"

	"github.com/clementus360/proxy-chat/database"
	"github.com/clementus360/proxy-chat/handlers"
	"github.com/clementus360/proxy-chat/websocket"

	"github.com/rs/cors"
)

func main() {
	// Initialize PostgreSQL & Redis
	database.InitPostgres()
	database.InitRedis()

	// Run database migrations
	database.RunMigrations()

	// Set up http routes
	http.HandleFunc("POST /api/users", handlers.CreateUser)   // POST /users
	http.HandleFunc("GET /api/users", handlers.GetUsers)      // GET /users/:lat/:long
	http.HandleFunc("PATCH /api/users", handlers.UpdateUser)  // PATCH /users
	http.HandleFunc("DELETE /api/users", handlers.DeleteUser) // DELETE /users

	http.HandleFunc("POST /api/groups", handlers.CreateGroup)    // POST /groups
	http.HandleFunc("GET /api/groups", handlers.GetGroups)       // GET /groups/:lat/:long
	http.HandleFunc("POST /api/groups/join", handlers.JoinGroup) // GET /group/:group_id

	http.HandleFunc("POST /api/messages", handlers.SendMessage) // POST /messages
	http.HandleFunc("GET /api/messages", handlers.GetMessages)  // GET /messages/:group_id

	http.HandleFunc("GET /ws", websocket.HandleWebSocket)

	// Set up CORS
	c := cors.AllowAll()
	handler := c.Handler(http.DefaultServeMux)

	log.Println("Proximity chat backend is running...")
	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
