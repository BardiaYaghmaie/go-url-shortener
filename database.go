package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB(filepath string) error {
	var err error
	db, err = sql.Open("sqlite3", filepath)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		short_code TEXT UNIQUE,
		long_url TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}
	log.Println("Database initialized successfully")
	return nil
}

func insertURL(shortCode, longURL string) error {
	_, err := db.Exec("INSERT INTO urls (short_code, long_url) VALUES (?, ?)", shortCode, longURL)
	return err
}

func getLongURL(shortCode string) (string, error) {
	var longURL string
	err := db.QueryRow("SELECT long_url FROM urls WHERE short_code = ?", shortCode).Scan(&longURL)
	if err != nil {
		return "", err
	}
	return longURL, nil
}