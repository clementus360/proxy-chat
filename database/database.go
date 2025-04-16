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
	dbURL := config.GetEnv("DATABASE_URL", "postgres://postgres:password@localhost:5432/proxy_chat")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}

	DB = pool

	log.Println("Connected to database")
}
