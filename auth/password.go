package auth

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), 14)
	return string(bytes), err
}

func CheckPassword(plain, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}
