package models

import "time"

type User struct {
	ID         int       `json:"id"`
	Username   string    `json:"username"`
	Image_url  string    `json:"image_url"`
	Latitude   float64   `json:"latitude"`
	Longitude  float64   `json:"longitude"`
	Visible    bool      `json:"visible"`
	Online     bool      `json:"online"`
	LastActive time.Time `json:"last_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type Group struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Image_url string    `json:"image_url"`
	CreatorID int       `json:"creator_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID         int       `json:"id"`
	Content    string    `json:"content"`
	GroupID    int       `json:"group_id"`
	SenderID   int       `json:"sender_id"`
	ReceiverID int       `json:"receiver_id"`
	CreatedAt  time.Time `json:"created_at"`
}
