package main

type URLRequest struct {
	ShortCode string `json:"short_code"`
	LongURL   string `json:"long_url"`
}

type URLResponse struct {
	ShortURL string `json:"short_url,omitempty"`
	LongURL  string `json:"long_url,omitempty"`
	Message  string `json:"message"`
}
