package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/JustZacca/jotaku/internal/auth"
	"github.com/JustZacca/jotaku/internal/db"
	"github.com/JustZacca/jotaku/internal/server"
)

func main() {
	// Configuration from environment
	port := getEnv("PORT", "5689")
	dbPath := getEnv("DB_PATH", "/data/notes.db")
	jwtSecret := getEnv("JWT_SECRET", "")

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// JWT expiration: 30 days
	jwtExpiration := 30 * 24 * time.Hour

	// Initialize database
	database, err := db.NewServerDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(jwtSecret, jwtExpiration)

	// Initialize server
	srv := server.New(database, jwtManager)

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Database: %s", dbPath)

	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
