package main

import (
	"math/rand"
	"net/url"
	"strings"
	"time"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateShortCode(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func isValidURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func isValidShortCode(code string) bool {
	if len(code) < 4 || len(code) > 20 {
		return false
	}
	for _, c := range code {
		if !strings.ContainsRune(charset, c) {
			return false
		}
	}
	return true
}