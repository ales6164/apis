package apis

import (
	"golang.org/x/crypto/bcrypt"
)

const COST = 14

func decrypt(hash []byte, password []byte) error {
	defer clear(password)
	return bcrypt.CompareHashAndPassword(hash, password)
}

func crypt(password []byte) ([]byte, error) {
	defer clear(password)
	return bcrypt.GenerateFromPassword(password, COST)
}

func clear(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = 0
	}
}
