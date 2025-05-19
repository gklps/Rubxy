package db

import (
	"database/sql"
	"errors"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Init(databaseURL string) {
	var err error
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Database not reachable: %v", err)
	}

	createTables()
}

func createTables() {
	createUsersTable()
	createRefreshTokensTable()
}

func createUsersTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	);`
	_, err := DB.Exec(query)
	if err != nil {
		log.Fatalf("Failed to create 'users' table: %v", err)
	}
}

func createRefreshTokensTable() {
	query := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		token TEXT PRIMARY KEY,
		username TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		revoked BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT NOW()
	);`
	_, err := DB.Exec(query)
	if err != nil {
		log.Fatalf("Failed to create 'refresh_tokens' table: %v", err)
	}
}

// SaveRefreshToken inserts a refresh token record into DB
func SaveRefreshToken(token, username string, expiresAt time.Time) error {
	query := `INSERT INTO refresh_tokens (token, username, expires_at) VALUES ($1, $2, $3)`
	_, err := DB.Exec(query, token, username, expiresAt)
	return err
}

// CheckRefreshTokenExists returns true if token exists and is valid (not revoked or expired)
func CheckRefreshTokenExists(token string) (bool, error) {
	var revoked bool
	var expiresAt time.Time

	query := `SELECT revoked, expires_at FROM refresh_tokens WHERE token = $1`
	err := DB.QueryRow(query, token).Scan(&revoked, &expiresAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if revoked || time.Now().After(expiresAt) {
		return false, nil
	}
	return true, nil
}

// RevokeRefreshToken marks the token as revoked
func RevokeRefreshToken(token string) error {
	query := `UPDATE refresh_tokens SET revoked = TRUE WHERE token = $1`
	res, err := DB.Exec(query, token)
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("token not found")
	}
	return nil
}

func IsRefreshTokenValid(token string) (bool, error) {
	var revoked bool
	var expiresAt time.Time

	query := `SELECT revoked, expires_at FROM refresh_tokens WHERE token = $1`
	err := DB.QueryRow(query, token).Scan(&revoked, &expiresAt)
	if err == sql.ErrNoRows {
		return false, nil // token not found
	}
	if err != nil {
		return false, err
	}
	if revoked || time.Now().After(expiresAt) {
		return false, nil
	}
	return true, nil
}
