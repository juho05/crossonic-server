package repos

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"

	"github.com/nullism/bqb"
	"golang.org/x/crypto/argon2"
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

func EncryptPassword(password string, key []byte) ([]byte, error) {
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(aesCipher)
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

func DecryptPassword(encryptedPassword, key []byte) (string, error) {
	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(aesCipher)
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

const argon2idTime = 1
const argon2idMemory = 3072
const argon2idThreads = 4

func HashPassword(password string) (string, error) {
	salt, err := generateRandomBytes(16)
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, 1, 3072, 4, 32)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, argon2idMemory, argon2idTime, argon2idThreads, b64Salt, b64Hash)

	return encodedHash, nil
}

func VerifyPassword(expected string, provided string) (bool, error) {
	parts := strings.Split(expected, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid expected hash")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}
	if version != argon2.Version {
		return false, fmt.Errorf("argon2id version mismatch: expected %d, got %d", argon2.Version, version)
	}

	var memory uint32
	var time uint32
	var threads uint8
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	hash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("invalid expected hash: %w", err)
	}

	providedHash := argon2.IDKey([]byte(provided), salt, time, memory, threads, uint32(len(hash)))

	return subtle.ConstantTimeCompare(hash, providedHash) == 1, nil
}

const (
	apiKeyChars = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ123456789"
)

func GenerateAPIKey() (string, error) {
	var builder strings.Builder
	for i := range 6 {
		if i > 0 {
			builder.WriteRune('-')
		}
		for range 5 {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(apiKeyChars))))
			if err != nil {
				return "", fmt.Errorf("get random char: %w", err)
			}
			builder.WriteByte(apiKeyChars[n.Int64()])
		}
	}
	return builder.String(), nil
}

func HashAPIKey(apiKey string) ([]byte, error) {
	return pbkdf2.Key(sha256.New, apiKey, []byte("crossonic-server"), 4096, 32)
}

func generateRandomBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}
