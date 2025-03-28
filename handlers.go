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

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var longURL string
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		var data struct {
			URL string `json:"url"`
		}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		longURL = data.URL
	} else {
		// Handle form data from HTMX
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		longURL = r.FormValue("url")
	}

	if !isValidURL(longURL) {
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<div class="alert alert-danger" role="alert">Invalid URL</div>`)
		} else {
			http.Error(w, "Invalid URL", http.StatusBadRequest)
		}
		return
	}

	var shortCode string
	for {
		shortCode = generateShortCode(6)
		err := insertURL(shortCode, longURL)
		if err == nil {
			break
		}
		if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.Code == sqlite3.ErrConstraint {
			continue
		}
		log.Println("Database error:", err)
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<div class="alert alert-danger" role="alert">Database error</div>`)
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	shortURL := fmt.Sprintf("http://%s/%s", r.Host, shortCode)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div class="alert alert-success" role="alert">Shortened URL: <a href="%s" class="alert-link">%s</a></div>`, shortURL, shortURL)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"short_url": shortURL})
	}
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		// Serve the HTML form
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>URL Shortener</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <script src="https://unpkg.com/htmx.org@1.9.6"></script>
</head>
<body class="container mt-5">
    <h1 class="text-center">URL Shortener</h1>
    <form hx-post="/shorten" hx-target="#result" hx-swap="innerHTML" hx-on::after-request="this.reset()" class="mt-3">
        <div class="input-group">
            <input type="url" name="url" class="form-control" placeholder="Enter your URL" required>
            <button type="submit" class="btn btn-primary">Shorten</button>
        </div>
    </form>
    <div id="result" class="mt-3"></div>
</body>
</html>
		`)
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