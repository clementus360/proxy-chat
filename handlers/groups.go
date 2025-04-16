package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/clementus360/proxy-chat/database"
	"github.com/clementus360/proxy-chat/models"
)

type GroupResponse struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Image_url  string `json:"image_url"`
	Creator_id int    `json:"creator_id"`
}

type GetGroupsResponse struct {
	Groups     []GroupResponse `json:"groups"`
	TotalCount int             `json:"total_count"`
	Radius     int             `json:"radius_km"`
}

func CreateGroup(w http.ResponseWriter, r *http.Request) {

	// Parse request body
	var group models.Group
	err := json.NewDecoder(r.Body).Decode(&group)
	if err != nil {
		http.Error(w, "Unable to parse request body", http.StatusBadRequest)
		log.Println("Error parsing group from request body:", err)
		return
	}

	// Create initials from group name with random background color
	if group.Image_url == "" {
		groupName := strings.ReplaceAll(group.Name, " ", "")
		backgroundColor := fmt.Sprintf("%06x", (int(group.CreatorID)*12345+100000)&0xB0B0B0)
		textColor := "ffffff"
		group.Image_url = fmt.Sprintf("https://ui-avatars.com/api/?name=%s&background=%s&color=%s&size=256", groupName, backgroundColor, textColor)
	}

	// Ensure location is valid and create a point from latitude and longitude
	location := fmt.Sprintf("POINT(%f %f)", group.Longitude, group.Latitude)

	// insert group into database
	query := "INSERT INTO chat_groups (name, creator_id, latitude, longitude, image_url, location) VALUES ($1, $2, $3, $4, $5, ST_GeographyFromText($6)) RETURNING id, created_at, image_url"
	err = database.DB.QueryRow(r.Context(), query, group.Name, group.CreatorID, group.Latitude, group.Longitude, group.Image_url, location).Scan(&group.ID, &group.CreatedAt, &group.Image_url)
	if err != nil {
		http.Error(w, "Unable to create group", http.StatusInternalServerError)
		log.Println("Error creating group:", err)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
	log.Println("Group created:", group)
}

func GetGroups(w http.ResponseWriter, r *http.Request) {

	// default search radius is 5km
	radius := 5
	if r.URL.Query().Get("radius") != "" {
		parsedRadius, err := strconv.Atoi(r.URL.Query().Get("radius"))
		if err != nil || radius <= 0 {
			http.Error(w, "Invalid radius", http.StatusBadRequest)
			log.Println("Error parsing radius:", err)
			return
		}
		radius = parsedRadius
	}

	// Parse latitude and longitude from query string
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

	fmt.Println("Latitude:", lat, "Longitude:", long)

	// ensure location is valid and create a point from latitude and longitude
	location := fmt.Sprintf("POINT(%f %f)", long, lat)

	// fetch groups within the search radius
	query := `SELECT id, name, image_url FROM chat_groups WHERE ST_DWithin(location, ST_GeographyFromText($1), $2 * 1000)`
	rows, err := database.DB.Query(r.Context(), query, location, radius)
	if err != nil {
		http.Error(w, "Unable to fetch groups", http.StatusInternalServerError)
		log.Println("Error fetching groups:", err)
		return
	}
	defer rows.Close()

	// parse groups
	var groups []GroupResponse
	for rows.Next() {
		var group GroupResponse
		err = rows.Scan(&group.ID, &group.Name, &group.Image_url)
		if err != nil {
			http.Error(w, "Unable to fetch groups", http.StatusInternalServerError)
			log.Println("Error fetching groups:", err)
			return
		}
		groups = append(groups, group)
	}

	// send response
	response := GetGroupsResponse{
		Groups:     groups,
		TotalCount: len(groups),
		Radius:     radius,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Println("Groups fetched:", response)
}

func JoinGroup(w http.ResponseWriter, r *http.Request) {

	var requestData struct {
		UserID  string `json:"user_id"`
		GroupID string `json:"group_id"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Unable to parse request body", http.StatusBadRequest)
		log.Println("Error parsing request body:", err)
		return
	}

	if requestData.UserID == "" || requestData.GroupID == "" {
		http.Error(w, "Missing user_id or group_id", http.StatusBadRequest)
		log.Println("Missing user_id or group_id")
		return
	}

	// add user to group in Redis
	groupKey := fmt.Sprintf("group:%s", requestData.GroupID)

	// Check if the user is already a member of the group
	isMember, err := database.RedisClient.SIsMember(r.Context(), groupKey, requestData.UserID).Result()
	if err != nil {
		http.Error(w, "Unable to check group membership", http.StatusInternalServerError)
		log.Println("Error checking group membership:", err)
		return
	}
	if isMember {
		http.Error(w, "User is already a member of the group", http.StatusBadRequest)
		log.Println("User", requestData.UserID, "is already a member of group", requestData.GroupID)
		return
	}

	// Add user to group in Redis
	err = database.RedisClient.SAdd(r.Context(), groupKey, requestData.UserID).Err()
	if err != nil {
		http.Error(w, "Unable to join group", http.StatusInternalServerError)
		log.Println("Error joining group:", err)
		return
	}

	log.Println("User", requestData.UserID, "joined group", requestData.GroupID)
	// send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`Joined group successfully`))
}
