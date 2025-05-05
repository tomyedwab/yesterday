package util

import (
	"crypto/rand"
	"fmt"
	"os"
)

var JWTSecretKeyPath = "/tmp/jwtsecret.key"

func LoadJWTSecretKey() ([]byte, error) {
	key, err := os.ReadFile(JWTSecretKeyPath)
	if err != nil {
		// If the file doesn't exist, generate a new key
		if os.IsNotExist(err) {
			b := make([]byte, 32)
			_, err := rand.Read(b)
			if err != nil {
				return nil, fmt.Errorf("failed to generate random JWT secret key: %w", err)
			}
			if err := os.WriteFile(JWTSecretKeyPath, b, 0600); err != nil {
				return nil, fmt.Errorf("failed to write JWT secret key: %w", err)
			}
			key = b
		} else {
			return nil, fmt.Errorf("failed to read JWT secret key: %w", err)
		}
	}
	return key, nil
}
