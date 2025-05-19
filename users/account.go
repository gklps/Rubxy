package users

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username     string
	PasswordHash string
}

var (
	users = make(map[string]*User)
	mu    sync.Mutex
)

func Register(username, password string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := users[username]; exists {
		return fmt.Errorf("user already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	users[username] = &User{
		Username:     username,
		PasswordHash: string(hash),
	}
	return nil
}

func Authenticate(username, password string) bool {
	mu.Lock()
	user, exists := users[username]
	mu.Unlock()

	if !exists {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}
