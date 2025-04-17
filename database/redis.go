package database

import (
	"context"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/clementus360/proxy-chat/config"
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

func InitRedis() {
	config.LoadEnv()

	rawURL := config.GetEnv("REDIS_URL", "localhost:6379")
	password := config.GetEnv("REDIS_PASSWORD", "")
	db := 0
	addr := rawURL

	// Check if REDIS_URL is a full URI like redis://...
	if u, err := url.Parse(rawURL); err == nil && u.Scheme != "" {
		addr = u.Host

		if u.User != nil {
			pw, _ := u.User.Password()
			if pw != "" {
				password = pw
			}
		}

		if u.Path != "" && u.Path != "/" {
			if dbNum, err := strconv.Atoi(u.Path[1:]); err == nil {
				db = dbNum
			}
		}
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Unable to connect to redis: %v\n", err)
	}

	log.Println("Connected to Redis at", addr)
}
