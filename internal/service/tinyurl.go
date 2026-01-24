package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	pb "tinyurl/proto/tinyurl/v1"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TinyURLService struct {
	pb.UnimplementedTinyURLServer
	rdb              *redis.Client
	serverURL        string
	exclusiveLinkExp int
}

func NewTinyURLService(rdb *redis.Client, serverURL string, exclusiveLinkExp int) *TinyURLService {
	return &TinyURLService{
		rdb:              rdb,
		serverURL:        serverURL,
		exclusiveLinkExp: exclusiveLinkExp,
	}
}

func (s *TinyURLService) Shorten(ctx context.Context, req *pb.ShortenRequest) (*pb.ShortenResponse, error) {
	start := time.Now()
	if req.LongUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "long_url is required")
	}

	var shortCode, shortURL string

	if req.ShortCode != "" {
		shortCode = req.ShortCode
		// check collision
		if _, err := s.rdb.Get(ctx, shortCode).Result(); err == nil {
			return nil, status.Error(codes.AlreadyExists, "Short code already exists. Try another one!")
		}
	} else {
		// Generate
		charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		code := make([]byte, 10)
		for i := range code {
			code[i] = charset[rand.Intn(len(charset))]
		}
		shortCode = string(code)
	}

	shortURL = fmt.Sprintf("%s/%s", s.serverURL, shortCode)

	// Save to Redis
	exp := time.Duration(s.exclusiveLinkExp) * time.Hour
	err := s.rdb.Set(ctx, shortCode, req.LongUrl, exp).Err()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to save to Redis: %v", err)
	}

	elapsed := time.Since(start)
	fmt.Printf("[DEBUG] Shorten processed in %s\n", elapsed)

	return &pb.ShortenResponse{
		ShortUrl:    shortURL,
		LongUrl:     req.LongUrl,
		Message:     fmt.Sprintf("Exclusive link will be expired in %d hours", s.exclusiveLinkExp),
		ElapsedTime: elapsed.String(),
	}, nil
}

func (s *TinyURLService) GetOriginal(ctx context.Context, req *pb.GetOriginalRequest) (*pb.GetOriginalResponse, error) {
	if req.ShortCode == "" {
		return nil, status.Error(codes.InvalidArgument, "short_code is required")
	}

	longURL, err := s.rdb.Get(ctx, req.ShortCode).Result()
	if err == redis.Nil {
		return nil, status.Error(codes.NotFound, "URL not found")
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "Redis error: %v", err)
	}

	return &pb.GetOriginalResponse{
		LongUrl: longURL,
	}, nil
}
