package users

import (
	"fmt"
	"rubxy/db"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       int
	Username string
}

var (
	users = make(map[string]*User)
	mu    sync.Mutex
)

func Register(username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("password hash error: %w", err)
	}

	_, err = db.DB.Exec("INSERT INTO users (username, password_hash) VALUES ($1, $2)", username, string(hash))
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	return nil
}

func Authenticate(username, password string) bool {
	var hashed string
	err := db.DB.QueryRow("SELECT password_hash FROM users WHERE username=$1", username).Scan(&hashed)
	if err != nil {
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	return err == nil
}
