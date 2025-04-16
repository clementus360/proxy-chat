package database

import (
	"context"
	"log"
)

// RunMigrations ensures tables are created if they don’t exist
func RunMigrations() {
	ctx := context.Background()

	migrations := []string{
		// Users Table
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			latitude DECIMAL(9,6),
			longitude DECIMAL(9,6),
			location GEOGRAPHY(POINT, 4326),
			visible BOOLEAN DEFAULT FALSE,
			online BOOLEAN DEFAULT FALSE,
			image_url VARCHAR(255),
			last_active TIMESTAMP DEFAULT NOW(),
			created_at TIMESTAMP DEFAULT NOW()
		);`,

		// Chat Groups Table
		`CREATE TABLE IF NOT EXISTS chat_groups (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			creator_id INT REFERENCES users(id) ON DELETE CASCADE,
			latitude DECIMAL(9,6) NOT NULL,
			longitude DECIMAL(9,6) NOT NULL,
			location GEOGRAPHY(POINT, 4326),
			image_url VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW()
		);`,

		// User-Group Membership Table
		`CREATE TABLE IF NOT EXISTS group_memberships (
			user_id INT REFERENCES users(id) ON DELETE CASCADE,
			group_id INT REFERENCES chat_groups(id) ON DELETE CASCADE,
			joined_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (user_id, group_id)
		);`,

		// Messages Table (Supports both Group & 1-on-1 chats)
		`CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			sender_id INT REFERENCES users(id) ON DELETE CASCADE,
			receiver_id INT REFERENCES users(id) ON DELETE CASCADE, -- Nullable for group chats
			group_id INT REFERENCES chat_groups(id) ON DELETE CASCADE, -- Nullable for 1-on-1 chats
			content TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			CHECK (
				(receiver_id IS NOT NULL AND group_id IS NULL) OR 
				(receiver_id IS NULL AND group_id IS NOT NULL) -- Ensures a message is either direct OR group-based
			)
		);`,
	}

	for _, query := range migrations {
		_, err := DB.Exec(ctx, query)
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	log.Println("Migrations applied successfully.")
}
