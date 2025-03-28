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

	var longURL, customCode string
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		var data struct {
			URL        string `json:"url"`
			CustomCode string `json:"custom_code"`
		}
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		longURL = data.URL
		customCode = data.CustomCode
	} else {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		longURL = r.FormValue("url")
		customCode = r.FormValue("custom_code")
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
	if customCode != "" {
		if !isValidShortCode(customCode) {
			errorMsg := "Invalid short code. Must be 4-20 alphanumeric characters."
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<div class="alert alert-danger" role="alert">%s</div>`, errorMsg)
			} else {
				http.Error(w, errorMsg, http.StatusBadRequest)
			}
			return
		}

		err := insertURL(customCode, longURL)
		if err != nil {
			if sqliteErr, ok := err.(sqlite3.Error); ok && sqliteErr.Code == sqlite3.ErrConstraint {
				errorMsg := "This code is already in use."
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("Content-Type", "text/html")
					fmt.Fprintf(w, `<div class="alert alert-danger" role="alert">%s</div>`, errorMsg)
				} else {
					http.Error(w, errorMsg, http.StatusConflict)
				}
				return
			}
			log.Println("Database error:", err)
			errorMsg := "Database error."
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprintf(w, `<div class="alert alert-danger" role="alert">%s</div>`, errorMsg)
			} else {
				http.Error(w, errorMsg, http.StatusInternalServerError)
			}
			return
		}
		shortCode = customCode
	} else {
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
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css" rel="stylesheet">
    <style>
        :root {
            --gradient-start: #6366f1;
            --gradient-end: #a855f7;
            --glass-bg: rgba(255, 255, 255, 0.15);
        }
        
        body {
            background: linear-gradient(135deg, var(--gradient-start), var(--gradient-end));
            min-height: 100vh;
            font-family: 'Segoe UI', sans-serif;
            backdrop-filter: blur(10px);
        }
        
        .glass-card {
            background: var(--glass-bg);
            backdrop-filter: blur(10px);
            border-radius: 1rem;
            border: 1px solid rgba(255, 255, 255, 0.2);
            box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
            padding: 2rem;
            margin: 2rem 0;
        }

        .form-control {
            background: rgba(255, 255, 255, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.2);
            color: white !important;
        }

        .form-control:focus {
            background: rgba(255, 255, 255, 0.2);
            border-color: white;
            box-shadow: none;
        }

        .input-group-text {
            background: rgba(255, 255, 255, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.2);
            color: white !important;
        }

        .alert {
            animation: slideIn 0.3s ease-out;
            border: none;
        }

        @keyframes slideIn {
            from { transform: translateY(-20px); opacity: 0; }
            to { transform: translateY(0); opacity: 1; }
        }

        .short-url-box {
            background: rgba(255, 255, 255, 0.1);
            border-radius: 0.5rem;
            padding: 1rem;
            margin-top: 1rem;
            transition: all 0.3s ease;
        }

        .short-url-box:hover {
            background: rgba(255, 255, 255, 0.2);
        }

        .btn-hover {
            transition: all 0.3s ease;
            position: relative;
            overflow: hidden;
        }

        .btn-hover:hover {
            transform: translateY(-2px);
        }

        .btn-hover::after {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(255, 255, 255, 0.1);
            opacity: 0;
            transition: opacity 0.3s ease;
        }

        .btn-hover:hover::after {
            opacity: 1;
        }
    </style>
</head>
<body class="container py-5">
    <div class="row justify-content-center">
        <div class="col-md-8 col-lg-6">
            <div class="glass-card">
                <h1 class="text-center mb-4 text-white fw-bold">
                    <i class="fas fa-link me-2"></i>Shorten URL
                </h1>
                
                <form hx-post="/shorten" hx-target="#result" hx-swap="innerHTML" 
                      hx-on::after-request="this.reset()">
                    <div class="input-group mb-3">
                        <span class="input-group-text">
                            <i class="fas fa-link"></i>
                        </span>
                        <input type="url" name="url" class="form-control form-control-lg" 
                               placeholder="Paste your long URL here" required>
                    </div>

                    <div class="input-group mb-4">
                        <span class="input-group-text">
                            <i class="fas fa-magic"></i>
                        </span>
                        <input type="text" name="custom_code" class="form-control form-control-lg" 
                               placeholder="Custom short code (optional)" 
                               pattern="[A-Za-z0-9]{4,20}" 
                               title="4 to 20 alphanumeric characters">
                    </div>

                    <button type="submit" class="btn btn-light btn-lg w-100 btn-hover">
                        <i class="fas fa-rocket me-2"></i>Shorten URL
                    </button>
                </form>

                <div id="result" class="mt-4"></div>
            </div>

            <div class="text-center mt-4 text-white opacity-75">
                <p class="small">
                    <i class="fas fa-info-circle me-2"></i>
                    Short links never expire. Custom codes are case-sensitive.
                </p>
            </div>
        </div>
    </div>

    <script>
        // Copy to clipboard functionality
        document.body.addEventListener('click', function(e) {
            if (e.target.classList.contains('copy-btn')) {
                const text = e.target.getAttribute('data-clipboard-text');
                navigator.clipboard.writeText(text).then(() => {
                    const alert = document.createElement('div');
                    alert.className = 'alert alert-info';
                    alert.innerHTML = 'Copied to clipboard!';
                    document.getElementById('result').appendChild(alert);
                    setTimeout(() => alert.remove(), 2000);
                });
            }
        });

        // Auto-focus on URL input
        document.querySelector('input[name="url"]').focus();
    </script>
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
