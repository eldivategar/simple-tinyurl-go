package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"time"
)

func TinyURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var req URLRequest
	var res URLResponse

	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error while reading request body!", http.StatusInternalServerError)
		return
	}

	cleanBody := invalidEscapeRx.ReplaceAll(b, []byte("$1"))
	if err := json.Unmarshal(cleanBody, &req); err != nil {
		http.Error(w, fmt.Sprintf("Error while parsing request body:  %v", err), http.StatusBadRequest)
		return
	}
	var shortURL, shortCode string

	if req.ShortCode != "" {
		shortCode = req.ShortCode
		shortURL = ServerURL + "/" + shortCode

		// check if short code is already in redis
		if _, err := rdb.Get(ctx, shortCode).Result(); err == nil {
			w.WriteHeader(http.StatusBadRequest)
			res = URLResponse{
				Message: "Try another one!",
			}
			json.NewEncoder(w).Encode(res)
			return
		}
	} else {
		shortURL, shortCode = GenerateShortURL(req.LongURL)
	}

	// save url in redis
	exp := time.Duration(ExlusiveLinkExp) * time.Hour
	err = rdb.Set(ctx, shortCode, req.LongURL, exp).Err()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error while saving URL in Redis!", http.StatusInternalServerError)
		return
	}

	res = URLResponse{
		ShortURL: shortURL,
		LongURL:  req.LongURL,
		Message:  fmt.Sprintf("Exclusive link will be expired in %d hours", ExlusiveLinkExp),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	shortURL := r.URL.Path
	shortCode := strings.TrimPrefix(shortURL, "/")

	longURL, err := GetLongURL(shortCode)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error while getting long URL!", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longURL, http.StatusSeeOther)
}
