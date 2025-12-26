package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"os"

	"github.com/redis/go-redis/v9"
)

var (
	ServerURL = "http://localhost:8080"
)

var rdb *redis.Client
var ctx = context.Background()

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL != "" {
		ServerURL = serverURL
	}

	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/tinyurl", TinyURLHandler)
	mux.HandleFunc("/", RedirectHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("Server starting on :8080")
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Error starting server:", err)
		}
	}
}
