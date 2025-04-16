package database

import (
	"context"
	"log"
	"time"

	"github.com/clementus360/proxy-chat/config"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	config.LoadEnv()
	redisURL := config.GetEnv("REDIS_URL", "localhost:6379")
	redisPassword := config.GetEnv("REDIS_PASSWORD", "")
	redisDB := 0

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Unable to connect to redis: %v\n", err)
	}

	log.Println("Connected to redis")
}
