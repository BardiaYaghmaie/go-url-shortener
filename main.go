package main

import (
	"log"
	"net/http"
)

func main() {
	err := initDB("urls.db")
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/shorten", shortenHandler)
	mux.HandleFunc("/", redirectHandler)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}