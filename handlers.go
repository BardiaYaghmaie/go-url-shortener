package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/mattn/go-sqlite3"
)

// shortenHandler handles POST requests to /shorten, generating a short URL.
func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		URL string `json:"url"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !isValidURL(data.URL) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	for {
		shortCode := generateShortCode(6)
		err = insertURL(shortCode, data.URL)
		if err == nil {
			shortURL := fmt.Sprintf("http://%s/%s", r.Host, shortCode)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"short_url": shortURL})
			return
		}
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.Code == sqlite3.ErrConstraint {
			// Unique constraint violation, try again
			continue
		}
		log.Println("Database error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
}

// redirectHandler handles GET requests to /:short_code, redirecting to the original URL.
func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		w.Write([]byte("URL Shortener Service\nPOST to /shorten with JSON body {\"url\": \"https://example.com\"} to get a short URL."))
		return
	}

	longURL, err := getLongURL(path)
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Println("Database error:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longURL, http.StatusMovedPermanently)
}