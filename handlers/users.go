package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/clementus360/proxy-chat/database"
	"github.com/clementus360/proxy-chat/models"
)

// Define a response struct that excludes latitude & longitude
type UserResponse struct {
	ID         int       `json:"id"`
	Username   string    `json:"username"`
	Image_url  string    `json:"image_url"`
	Visible    bool      `json:"visible"`
	Online     bool      `json:"online"`
	LastActive time.Time `json:"last_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// Response struct for GetUsers API
type GetUsersResponse struct {
	Users      []UserResponse `json:"users"`
	TotalCount int            `json:"total_count"`
	Radius     int            `json:"radius_km"`
}

func CreateUser(w http.ResponseWriter, r *http.Request) {

	// Parse request body
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, "Unable to parse request body", http.StatusBadRequest)
		log.Println("Error parsing user from request body:", err)
		return
	}

	// Create initials from username
	if user.Image_url == "" {
		userName := strings.ReplaceAll(user.Username, " ", "")
		backgroundColor := fmt.Sprintf("%06x", (int(user.ID)*12345+100000)&0xB0B0B0)
		textColor := "ffffff"
		user.Image_url = fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=%s&color=%s&size=256", userName, backgroundColor, textColor)
	}

	// Ensure location is valid and create a point from latitude and longitude
	location := fmt.Sprintf("POINT(%f %f)", user.Longitude, user.Latitude)

	// insert user into database
	query := "INSERT INTO users (username, latitude, longitude, image_url, location) VALUES ($1, $2, $3, $4, ST_GeographyFromText($5)) RETURNING id, created_at"
	err = database.DB.QueryRow(r.Context(), query, user.Username, user.Latitude, user.Longitude, user.Image_url, location).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		http.Error(w, "Unable to create user", http.StatusInternalServerError)
		log.Println("Error creating user:", err)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	log.Println("User created:", user)
}

func GetUsers(w http.ResponseWriter, r *http.Request) {

	// Parse user id from URL
	userID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid user id", http.StatusBadRequest)
		log.Println("Error parsing user id:", err)
		return
	}

	// Default search radius is 5km
	radius := 5
	if r.URL.Query().Get("radius") != "" {
		parsedRadius, err := strconv.Atoi(r.URL.Query().Get("radius"))
		if err != nil || parsedRadius <= 0 {
			http.Error(w, "Invalid radius", http.StatusBadRequest)
			log.Println("Error parsing radius:", err)
			return
		}
		radius = parsedRadius
	}

	// Parse latitude and longitude from URL
	lat, err := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	if err != nil {
		http.Error(w, "Invalid latitude", http.StatusBadRequest)
		log.Println("Error parsing latitude:", err)
		return
	}

	long, err := strconv.ParseFloat(r.URL.Query().Get("long"), 64)
	if err != nil {
		http.Error(w, "Invalid longitude", http.StatusBadRequest)
		log.Println("Error parsing longitude:", err)
		return
	}

	// Ensure location is valid and create a point from latitude and longitude
	location := fmt.Sprintf("POINT(%f %f)", long, lat)

	query := `
		SELECT id, username, image_url, visible, online, last_active, created_at 
		FROM users 
		WHERE ST_DWithin(
		location, ST_GeographyFromText($1), $2 * 1000
		) AND visible = TRUE AND id != $3 AND online = TRUE;`

	rows, err := database.DB.Query(r.Context(), query, location, radius, userID)
	if err != nil {
		http.Error(w, "Unable to fetch users", http.StatusInternalServerError)
		log.Println("Error fetching users:", err)
		return
	}
	defer rows.Close()

	// Parse rows into user struct
	var users []UserResponse
	for rows.Next() {
		var user UserResponse
		err = rows.Scan(&user.ID, &user.Username, &user.Image_url, &user.Visible, &user.Online, &user.LastActive, &user.CreatedAt)
		if err != nil {
			http.Error(w, "Unable to fetch users", http.StatusInternalServerError)
			log.Println("Error fetching users:", err)
			return
		}
		users = append(users, user)
	}

	// Create response
	response := GetUsersResponse{
		Users:      users,
		TotalCount: len(users),
		Radius:     radius,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Println("Users fetched:", users)
}

func UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var updates map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&updates)
	if err != nil {
		http.Error(w, "Unable to parse request body", http.StatusBadRequest)
		log.Println("Error parsing user updates from request body:", err)
		return
	}

	// Parse user id from URL
	userID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid user id", http.StatusBadRequest)
		log.Println("Error parsing user id:", err)
		return
	}

	// Build query string
	var queryParts []string
	var queryParams []interface{}
	argIndex := 1

	// handle updatable fields
	if username, exists := updates["username"]; exists {
		queryParts = append(queryParts, fmt.Sprintf("username = $%d", argIndex))
		queryParams = append(queryParams, username)
		argIndex++
	}

	if imageURL, exists := updates["image_url"]; exists {
		queryParts = append(queryParts, fmt.Sprintf("image_url = $%d", argIndex))
		queryParams = append(queryParams, imageURL)
		argIndex++
	}

	if latitude, exists := updates["latitude"]; exists {
		queryParts = append(queryParts, fmt.Sprintf("latitude = $%d", argIndex))
		queryParams = append(queryParams, latitude)
		argIndex++
	}

	if longitude, exists := updates["longitude"]; exists {
		queryParts = append(queryParts, fmt.Sprintf("longitude = $%d", argIndex))
		queryParams = append(queryParams, longitude)
		argIndex++
	}

	if visible, exists := updates["visible"]; exists {
		queryParts = append(queryParts, fmt.Sprintf("visible = $%d", argIndex))
		queryParams = append(queryParams, visible)
		argIndex++
	}

	// Ensure location is valid and create a point from latitude and longitude
	if latitude, exists := updates["latitude"]; exists {
		if longitude, exists := updates["longitude"]; exists {
			location := fmt.Sprintf("POINT(%f %f)", longitude, latitude)
			queryParts = append(queryParts, fmt.Sprintf("location = ST_GeographyFromText($%d)", argIndex))
			queryParams = append(queryParams, location)
			argIndex++
		}
	}

	// if no updatable fields are provided
	if len(queryParts) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		log.Println("No fields to update")
		return
	}

	// Finalize query string
	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = $%d RETURNING id, username, image_url, visible, last_active, created_at;`, strings.Join(queryParts, ", "), argIndex)

	queryParams = append(queryParams, userID)

	fmt.Println(query)
	fmt.Println(queryParams)

	// Update user in database
	var user models.User
	err = database.DB.QueryRow(r.Context(), query, queryParams...).Scan(&user.ID, &user.Username, &user.Image_url, &user.Visible, &user.LastActive, &user.CreatedAt)
	if err != nil {
		http.Error(w, "Unable to update user", http.StatusInternalServerError)
		log.Println("Error updating user:", err)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
	log.Println("User updated:", user)
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Parse user id from URL
	userID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, "Invalid user id", http.StatusBadRequest)
		log.Println("Error parsing user id:", err)
		return
	}

	// delete user from database
	query := "DELETE FROM users WHERE id = $1"
	_, err = database.DB.Exec(r.Context(), query, userID)
	if err != nil {
		http.Error(w, "Unable to delete user", http.StatusInternalServerError)
		log.Println("Error deleting user:", err)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "User  deleted successfully"}`))
	log.Println("User deleted:", userID)
}
