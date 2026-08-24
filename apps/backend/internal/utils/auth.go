package utils

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	iterations           = 3
	memory               = 64 * 1024
	parallelism          = 4
	keyLen               = 32
	PasswordAuthProvider = "password"
	GoogleAuthProvider   = "google"
)

var (
	ErrInvalidHash         = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
	ErrIncompatibleVariant = errors.New("incompatible argon2 variant (expected argon2id)")
)

type argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func generateRandomSalt() []byte {
	salt := make([]byte, 16)
	n, _ := rand.Read(salt)
	return salt[:n]
}

// encode hash in base64 and return encoded string
func encodeHash(salt, hashedPassword []byte) string {
	saltBase64 := base64.RawStdEncoding.EncodeToString(salt)
	passwordBase64 := base64.RawStdEncoding.EncodeToString(hashedPassword)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memory, iterations, parallelism, saltBase64, passwordBase64)
}

// DecodeHash extracts the parameters, salt, and raw hash from an encoded Argon2 string.
func DecodeHash(encodedHash string) (p *argon2Params, salt, hash []byte, err error) {
	// 1. Split the string.
	// A valid string starts with '$', so the first element will be an empty string.
	// Expected parts: "", "argon2id", "v=19", "m=...,t=...,p=...", "salt", "hash"
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	// 2. Enforce the Algorithm Variant
	if vals[1] != "argon2id" {
		return nil, nil, nil, ErrIncompatibleVariant
	}

	// 3. Enforce the Version
	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil || version != 19 {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	// 4. Extract Configuration Parameters
	p = &argon2Params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Iterations, &p.Parallelism)
	if err != nil {
		return nil, nil, nil, ErrInvalidHash
	}

	// 5. Decode the Salt
	// Argon2 strings use Raw Base64 (no padding '=' at the end).
	// We use Strict() to reject malformed base64 to prevent silent corruption.
	salt, err = base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode salt: %w", err)
	}
	p.SaltLength = uint32(len(salt))

	// 6. Decode the Hash
	hash, err = base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode hash: %w", err)
	}
	p.KeyLength = uint32(len(hash))

	return p, salt, hash, nil
}

// create argon2 hashed password
func CreatePasswordHash(password string) string {
	salt := generateRandomSalt()
	hashedPassword := argon2.IDKey([]byte(password), []byte(salt), iterations, memory, parallelism, keyLen)

	return encodeHash(salt, hashedPassword)
}

// compare argon2 hashed password and return true if they match
func ComparePassword(password, hashedPassword string) bool {
	argonParams, salt, h, err := DecodeHash(hashedPassword)
	if err != nil {
		return false
	}
	hash := argon2.IDKey([]byte(password), []byte(salt), argonParams.Iterations, argonParams.Memory, argonParams.Parallelism, argonParams.KeyLength)
	return string(hash) == string(h)
}
