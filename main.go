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
	ServerURL = "http://localhost:8000"
)

var rdb *redis.Client
var ctx = context.Background()

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")
	if redisPassword == "" {
		redisPassword = ""
	}

	serverURL := os.Getenv("SERVER_URL")
	if serverURL != "" {
		ServerURL = serverURL
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Println("Error connecting to Redis:", err)
	} else {
		fmt.Println("Connected to Redis")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/tinyurl", TinyURLHandler)
	mux.HandleFunc("/", RedirectHandler)

	server := &http.Server{
		Addr:    ":8000",
		Handler: corsMiddleware(mux),
	}

	fmt.Println("Server starting on :8000")
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Error starting server:", err)
		}
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set Header CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// Handle Preflight (OPTIONS)
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
