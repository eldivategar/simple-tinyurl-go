package main

type URLRequest struct {
	LongURL string `json:"long_url"`
}

type URLResponse struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
	Message  string `json:"message"`
}
