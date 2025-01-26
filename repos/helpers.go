package repos

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/juho05/crossonic-server/config"
	"github.com/nullism/bqb"
)

type Paginate struct {
	Offset int
	Limit  *int
}

func (p Paginate) Apply(q *bqb.Query) {
	if p.Offset > 0 {
		q.Space("OFFSET ?", p.Offset)
	}
	if p.Limit != nil {
		q.Space("LIMIT ?", max(*p.Limit, 0))
	}
}

type Optional[T any] struct {
	value    T
	hasValue bool
}

func NewOptional[T any](value T, hasValue bool) Optional[T] {
	return Optional[T]{
		value:    value,
		hasValue: hasValue,
	}
}

func NewOptionalFull[T any](value T) Optional[T] {
	return Optional[T]{
		value:    value,
		hasValue: true,
	}
}

func NewOptionalEmpty[T any]() Optional[T] {
	return Optional[T]{
		hasValue: false,
	}
}

func (o Optional[T]) HasValue() bool {
	return o.hasValue
}

func (o Optional[T]) Get() any {
	if !o.HasValue() {
		return nil
	}
	return o.value
}

type OptionalGetter interface {
	HasValue() bool
	Get() any
}

func EncryptPassword(password string) ([]byte, error) {
	aes, err := aes.NewCipher(config.PasswordEncryptionKey())
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return gcm.Seal(nonce, nonce, []byte(password), nil), nil
}

func DecryptPassword(encryptedPassword []byte) (string, error) {
	aes, err := aes.NewCipher(config.PasswordEncryptionKey())
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(aes)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := encryptedPassword[:nonceSize], encryptedPassword[nonceSize:]
	password, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(password), nil
}
