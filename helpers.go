package main

import (
	"errors"
	"math/rand"
	"net/http"
	"regexp"
	"strings"

	"github.com/redis/go-redis/v9"
)

var invalidEscapeRx = regexp.MustCompile(`\\([^"\\/bfnrtu])`)

func GenerateShortURL(longURL string) (string, string) {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 10)

	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	url := ServerURL + "/" + string(code)
	return string(url), string(code)
}

func GetLongURL(shortCode string) (string, error) {
	originalURL, err := rdb.Get(ctx, shortCode).Result()
	if err == redis.Nil {
		return "", errors.New("URL not found")
	} else if err != nil {
		return "", err
	}

	return originalURL, nil
}

func getRealIP(r *http.Request) string {
	xfwd := r.Header.Get("X-Forwarded-For")
	if xfwd != "" {
		ips := strings.Split(xfwd, ",")
		return strings.TrimSpace(ips[0])
	}
	return r.RemoteAddr
}
