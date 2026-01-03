package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var (
	ServerURL            = "http://localhost:7860"
	RateLimitMax         = 10
	RateLimitWindows     = 1 * time.Minute
	ExlusiveLinkExp  int = 24 // in hours
)

var rdb *redis.Client
var ctx = context.Background()

func main() {
	// load env
	godotenv.Load()

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

	if exclusiveLinkExp := os.Getenv("EXCLUSIVE_LINK_EXP"); exclusiveLinkExp != "" {
		ExlusiveLinkExp, _ = strconv.Atoi(exclusiveLinkExp)
	}

	redisOptions := &redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       0,
	}

	if strings.Contains(redisAddr, "upstash.io") {
		redisOptions.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb = redis.NewClient(redisOptions)

	if err := rdb.Ping(ctx).Err(); err != nil {
		fmt.Println("Error connecting to Redis:", err)
	} else {
		fmt.Println("Connected to Redis")
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/tinyurl", TinyURLHandler)
	mux.HandleFunc("/", RedirectHandler)

	server := &http.Server{
		Addr:    ":7860",
		Handler: corsMiddleware(rateLimitMiddleware(mux)),
	}

	// instance of scheduller
	scheduller := NewScheduller()
	defer scheduller.Stop()

	// heartbeat every 1 day
	scheduller.AddFunc("@daily", func() {
		fmt.Println("Running heartbeat job")
	})

	// start the scheduller
	go scheduller.Start()

	fmt.Println("Server starting on :7860")
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

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If rate limit is exceeded, return 429 (Too Many Requests)
		if r.URL.Path == "/tinyurl" && r.Method == "POST" {
			ip := getRealIP(r)
			key := "rate_limit:" + ip

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				fmt.Println("Redis error:", err)
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				rdb.Expire(ctx, key, RateLimitWindows)
			}

			if count > int64(RateLimitMax) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Too many requests",
					"message": "Try again after 1 minute.",
				})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
