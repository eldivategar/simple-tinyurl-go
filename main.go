package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	"tinyurl/internal/service"
	pb "tinyurl/proto/tinyurl/v1"
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

	// scheduler
	scheduller := NewScheduller()
	defer scheduller.Stop()
	scheduller.AddFunc("@daily", func() {
		fmt.Println("Running heartbeat job")
	})
	go scheduller.Start()

	// --- gRPC Server Setup ---
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("Failed to listen on :50051: %v\n", err)
		return
	}

	grpcServer := grpc.NewServer()
	tinyURLService := service.NewTinyURLService(rdb, ServerURL, ExlusiveLinkExp)
	pb.RegisterTinyURLServer(grpcServer, tinyURLService)

	// Register reflection service
	reflection.Register(grpcServer)

	go func() {
		fmt.Println("Starting gRPC server on :50051")
		if err := grpcServer.Serve(lis); err != nil {
			fmt.Printf("Failed to serve gRPC: %v\n", err)
		}
	}()

	// --- gRPC Gateway Setup ---
	// Create a client connection to the gRPC server we just started
	// This is used by the Gateway to translate HTTP -> gRPC
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Printf("Failed to dial server: %v\n", err)
		return
	}
	defer conn.Close()

	gwmux := runtime.NewServeMux()
	// Register the handler (translates REST to gRPC)
	err = pb.RegisterTinyURLHandler(ctx, gwmux, conn)
	if err != nil {
		fmt.Printf("Failed to register gateway: %v\n", err)
		return
	}

	// --- Main HTTP Server (Port 7860) ---
	// Multiplex between Gateway (API) and Native Handlers (Redirect, Static)
	mux := http.NewServeMux()

	// 1. Serving index.html on root if not a short code?
	// The problem is "/" matches everything in Go 1.22 mux unless updated, but let's assume standard behavior.
	// We need to distinguish between GET / (index.html) and GET /SHORTCODE (redirect).

	// Create a client for the redirect handler to use
	grpcClient := pb.NewTinyURLClient(conn)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If path is exactly "/", serve index.html
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "index.html")
			return
		}

		// Check if it is /tinyurl (Gateway should handle this, but mux logic is specific)
		// Since we registered gateway separately, we need to route specific paths to it.
		// The gateway handles /tinyurl (POST) and /v1/url/{code} (GET).
		if strings.HasPrefix(r.URL.Path, "/tinyurl") || strings.HasPrefix(r.URL.Path, "/v1/url") {
			gwmux.ServeHTTP(w, r)
			return
		}

		// Otherwise, treat as Short Code Redirect
		// GET /{shortCode}
		shortCode := strings.TrimPrefix(r.URL.Path, "/")
		if shortCode == "" {
			http.ServeFile(w, r, "index.html") // Fallback
			return
		}

		// Call gRPC GetOriginal
		resp, err := grpcClient.GetOriginal(ctx, &pb.GetOriginalRequest{ShortCode: shortCode})
		if err != nil {
			// Handle error (Not Found, etc)
			// gRPC error codes need to be mapped if we want specific HTTP statuses,
			// but for now 404 or 500 is fine.
			fmt.Printf("Redirect error for %s: %v\n", shortCode, err)
			http.Error(w, "URL not found", http.StatusNotFound)
			return
		}

		http.Redirect(w, r, resp.LongUrl, http.StatusSeeOther)
	})

	server := &http.Server{
		Addr:    ":7860",
		Handler: corsMiddleware(rateLimitMiddleware(mux)),
	}

	fmt.Println("Server starting on :7860 (HTTP Gateway + Redirect)")
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Error starting server:", err)
		}
	}
}

// Keeping valid middlewares
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Updating Rate Limit to use new logic or just same path check
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Update path check: Gateway exposes /tinyurl
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

func getRealIP(r *http.Request) string {
	xfwd := r.Header.Get("X-Forwarded-For")
	if xfwd != "" {
		ips := strings.Split(xfwd, ",")
		return strings.TrimSpace(ips[0])
	}
	return r.RemoteAddr
}
