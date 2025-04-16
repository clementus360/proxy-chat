package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/clementus360/proxy-chat/database"
	"github.com/clementus360/proxy-chat/models"
)

func SendMessage(w http.ResponseWriter, r *http.Request) {

	// Parse request body
	var message models.Message
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, "Unable to parse request body", http.StatusBadRequest)
		log.Println("Error parsing message from request body:", err)
		return
	}

	// insert message into database
	query := "INSERT INTO messages (group_id, receiver_id, sender_id, content) VALUES ($1, $2, $3, $4) RETURNING id, created_at"
	err = database.DB.QueryRow(r.Context(), query, message.GroupID, message.ReceiverID, message.SenderID, message.Content).Scan(&message.ID, &message.CreatedAt)
	if err != nil {
		http.Error(w, "Unable to send message", http.StatusInternalServerError)
		log.Println("Error sending message:", err)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
	log.Println("Message sent:", message)
}

func GetMessages(w http.ResponseWriter, r *http.Request) {
	// Parse group id from query string
	groupId := r.URL.Query().Get("group_id")
	var messages []models.Message

	// If group_id is provided, fetch group messages
	if groupId != "" {
		groupIdInt, err := strconv.Atoi(groupId)
		if err != nil {
			http.Error(w, "Invalid group id", http.StatusBadRequest)
			log.Println("Error parsing group id:", err)
			return
		}

		// Fetch group messages
		query := "SELECT id, group_id, sender_id, content, created_at FROM messages WHERE group_id = $1"
		rows, err := database.DB.Query(r.Context(), query, groupIdInt)
		if err != nil {
			http.Error(w, "Unable to fetch group messages", http.StatusInternalServerError)
			log.Println("Error fetching group messages:", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var message models.Message
			err = rows.Scan(&message.ID, &message.GroupID, &message.SenderID, &message.Content, &message.CreatedAt)
			if err != nil {
				http.Error(w, "Unable to fetch group messages", http.StatusInternalServerError)
				log.Println("Error fetching group messages:", err)
				return
			}
			messages = append(messages, message)
		}
	} else {
		// Fetch one-on-one messages
		// Example: Get messages for a user (both sent and received)
		userId := r.URL.Query().Get("user_id")
		if userId == "" {
			http.Error(w, "Missing user_id for one-on-one messages", http.StatusBadRequest)
			return
		}

		userIdInt, err := strconv.Atoi(userId)
		if err != nil {
			http.Error(w, "Invalid user id", http.StatusBadRequest)
			log.Println("Error parsing user id:", err)
			return
		}

		// Fetch one-on-one messages (sent or received)
		query := `SELECT id, sender_id, receiver_id, group_id, content, created_at 
		          FROM messages 
		          WHERE (sender_id = $1 AND receiver_id IS NOT NULL) 
		             OR (receiver_id = $1 AND sender_id IS NOT NULL)`
		rows, err := database.DB.Query(r.Context(), query, userIdInt)
		if err != nil {
			http.Error(w, "Unable to fetch one-on-one messages", http.StatusInternalServerError)
			log.Println("Error fetching one-on-one messages:", err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var message models.Message
			err = rows.Scan(&message.ID, &message.SenderID, &message.ReceiverID, &message.GroupID, &message.Content, &message.CreatedAt)
			if err != nil {
				http.Error(w, "Unable to fetch one-on-one messages", http.StatusInternalServerError)
				log.Println("Error fetching one-on-one messages:", err)
				return
			}
			messages = append(messages, message)
		}
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
	log.Println("Messages fetched:", messages)
}
