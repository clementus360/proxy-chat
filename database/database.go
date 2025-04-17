package database

import (
	"context"
	"log"
	"time"

	"github.com/clementus360/proxy-chat/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitPostgres() {
	config.LoadEnv()
	dbURL := config.GetEnv("DATABASE_URL", "postgres://user:password@localhost:5432/proximity_chat")

	var pool *pgxpool.Pool
	var err error

	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pool, err = pgxpool.New(ctx, dbURL)
		if err == nil {
			// Try to ping the DB to ensure it's ready
			if err = pool.Ping(ctx); err == nil {
				break
			}
		}

		log.Printf("Attempt %d: Could not connect to Postgres. Retrying in 2s...\n", i)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("Failed to connect to Postgres after %d attempts: %v", maxRetries, err)
	}

	DB = pool
	log.Println("Connected to Postgres")
}
