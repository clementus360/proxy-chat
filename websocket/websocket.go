package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"fmt"

	"github.com/clementus360/proxy-chat/database"
	"github.com/gorilla/websocket"
)

var (
	ctx         = context.Background()
	userSockets = make(map[string]*websocket.Conn)
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// CheckOrigin is an optional function that can be used to check the origin of the request
		return true
	},
	EnableCompression: false, // Disable compression for now
	HandshakeTimeout:  10 * time.Second,
}

type WsMessage struct {
	Type       string    `json:"type"`
	GroupID    int       `json:"group_id,omitempty"`
	SenderID   int       `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	ReceiverID int       `json:"receiver_id,omitempty"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

func StoreWSConnection(userID string, conn *websocket.Conn) error {
	userSockets[userID] = conn
	return database.RedisClient.HSet(ctx, "active_users", userID, conn.RemoteAddr().String()).Err()
}

func RemoveWSConnection(userID string) error {
	delete(userSockets, userID)
	return database.RedisClient.HDel(ctx, "active_users", userID).Err()
}

func GetWSConnection(userID string) (*websocket.Conn, bool) {
	conn, exists := userSockets[userID]
	return conn, exists
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	err = StoreWSConnection(userID, conn)
	if err != nil {
		log.Printf("Error storing websocket connection: %v", err)
		return
	}

	// Set the user as online in postgres
	_, err = database.DB.Exec(ctx, "UPDATE users SET online = true WHERE id = $1", userID)
	if err != nil {
		log.Printf("Error setting user %s as online: %v", userID, err)
		return
	}

	defer func() {
		err := RemoveWSConnection(userID)
		if err != nil {
			log.Printf("Error removing websocket connection: %v", err)
		}

		// Set the user as offline in postgres
		_, err = database.DB.Exec(ctx, "UPDATE users SET online = false WHERE id = $1", userID)
		if err != nil {
			log.Printf("Error setting user %s as offline: %v", userID, err)
		}

		// Close the connection
		conn.Close()
		log.Printf("WebSocket connection closed for user %s", userID)
	}()

	// Fetch and send stored messages for offline users (RIGHT AFTER STORING CONNECTION)
	offlineKey := fmt.Sprintf("offline:%s", userID)
	messages, err := database.RedisClient.LRange(ctx, offlineKey, 0, -1).Result()
	if err != nil {
		log.Printf("Error fetching stored messages for user %s: %v", userID, err)
	} else {
		for _, msgStr := range messages {
			var storedMsg WsMessage
			if err := json.Unmarshal([]byte(msgStr), &storedMsg); err == nil {
				err = conn.WriteJSON(storedMsg)
				if err != nil {
					log.Printf("Error sending stored message to user %s: %v", userID, err)
				}
			}
		}
		// Clear stored messages after delivery
		database.RedisClient.Del(ctx, offlineKey)
	}

	for {
		var msg WsMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		// handle one to one messages
		if msg.ReceiverID != 0 {
			recieverConn, exists := GetWSConnection(fmt.Sprint(msg.ReceiverID))
			if exists {
				err = recieverConn.WriteJSON(msg)
				if err != nil {
					log.Printf("Error sending message to user %d: %v", msg.ReceiverID, err)
				}
			} else {
				// Store the message in Redis for offline users
				msgJSON, err := json.Marshal(msg)
				if err != nil {
					log.Printf("Error marshaling message: %v", err)
					continue
				}

				// Store the message in Redis with a TTL of 1 hour
				offlineKey := fmt.Sprintf("offline:%d", msg.ReceiverID)
				err = database.RedisClient.LPush(ctx, offlineKey, msgJSON).Err()
				if err != nil {
					log.Printf("Error storing offline message for user %d: %v", msg.ReceiverID, err)
					continue
				}

				// Set a TTL for the offline message
				err = database.RedisClient.Expire(ctx, offlineKey, time.Hour).Err()
				if err != nil {
					log.Printf("Error setting TTL for offline message for user %d: %v", msg.ReceiverID, err)
					continue
				}
				log.Printf("Stored offline message for user %d", msg.ReceiverID)

				// publish the message to the receiver's channel
				err = database.RedisClient.Publish(ctx, fmt.Sprintf("ws:%d", msg.ReceiverID), msgJSON).Err()
				if err != nil {
					log.Printf("Error publishing message to user %d: %v", msg.ReceiverID, err)
				}
			}
		}

		if msg.GroupID != 0 {
			// Handle group messages
			groupKey := fmt.Sprintf("group:%d", msg.GroupID)

			groupMembers, err := database.RedisClient.SMembers(ctx, groupKey).Result()
			if err != nil {
				log.Printf("Error fetching group members: %v", err)
				continue
			}

			for _, userID := range groupMembers {
				if userID == fmt.Sprintf("%d", msg.SenderID) {
					continue // Skip sending the message to the sender
				}
				receiverConn, exists := GetWSConnection(userID)
				if exists {
					err := receiverConn.WriteJSON(msg)
					if err != nil {
						log.Printf("Error delivering message to group member %s: %v", userID, err)
						continue
					}
				} else {
					// store the message in Redis for offline users
					msgJSON, err := json.Marshal(msg)
					if err != nil {
						log.Printf("Error marshaling message for group %d: %v", msg.GroupID, err)
						continue
					}

					// Store the message in Redis with a TTL of 1 hour
					offlineKey := fmt.Sprintf("offline:%s", userID)
					err = database.RedisClient.LPush(ctx, offlineKey, msgJSON).Err()
					if err != nil {
						log.Printf("Error storing offline message for group %d: %v", msg.GroupID, err)
						continue
					}
					// Set a TTL for the offline message
					err = database.RedisClient.Expire(ctx, offlineKey, time.Hour).Err()
					if err != nil {
						log.Printf("Error setting TTL for offline message for group %d: %v", msg.GroupID, err)
						continue
					}
					log.Printf("Stored offline message for group %d", msg.GroupID)
					// publish the message to the receiver's channel
					err = database.RedisClient.Publish(ctx, fmt.Sprintf("ws:%s", userID), msgJSON).Err()
					if err != nil {
						log.Printf("Error publishing message to group %d member %v: %v", msg.GroupID, userID, err)
					}
				}
			}
		}
	}
}

func SubscribeToMessages() {
	pubsub := database.RedisClient.Subscribe(ctx, "ws:*") // Subscribe to all user channels
	ch := pubsub.Channel()
	for msg := range ch {
		var wsMsg WsMessage
		err := json.Unmarshal([]byte(msg.Payload), &wsMsg)
		if err != nil {
			log.Printf("Error decoding WebSocket message: %v", err)
			continue
		}

		// Deliver the message to the connected client (if they are online)
		receiverConn, exists := GetWSConnection(fmt.Sprint(wsMsg.ReceiverID))
		if exists {
			err := receiverConn.WriteJSON(wsMsg)
			if err != nil {
				log.Printf("Error delivering message to user %d: %v", wsMsg.ReceiverID, err)
			}
		}
	}
}
