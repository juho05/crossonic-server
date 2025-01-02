package sqlc

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/juho05/crossonic-server/config"
)

func EncryptPassword(password string) ([]byte, error) {
	aes, err := aes.NewCipher(config.PasswordEncryptionKey())
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}
	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("encrypt password: generate nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, []byte(password), nil), nil
}

func DecryptPassword(encryptedPassword []byte) (string, error) {
	aes, err := aes.NewCipher(config.PasswordEncryptionKey())
	if err != nil {
		return "", fmt.Errorf("decrypt password: %w", err)
	}
	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return "", fmt.Errorf("decrypt password: %w", err)
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := encryptedPassword[:nonceSize], encryptedPassword[nonceSize:]
	password, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt password: %w", err)
	}
	return string(password), nil
}
