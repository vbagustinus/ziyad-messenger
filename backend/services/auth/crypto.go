package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

type PasswordConfig struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

var defaultConfig = &PasswordConfig{
	time:    1,
	memory:  64 * 1024,
	threads: 4,
	keyLen:  32,
}

// HashPassword generates an Argon2id hash of the password.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, defaultConfig.time, defaultConfig.memory, defaultConfig.threads, defaultConfig.keyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, defaultConfig.memory, defaultConfig.time, defaultConfig.threads, b64Salt, b64Hash)

	return encodedHash, nil
}

// VerifyPassword checks if the provided password matches the encoded hash.
func VerifyPassword(password, encodedHash string) (bool, error) {
	// Check if it's a bcrypt hash (used by admin-service)
	if strings.HasPrefix(encodedHash, "$2a$") || strings.HasPrefix(encodedHash, "$2b$") || strings.HasPrefix(encodedHash, "$2y$") {
		err := bcrypt.CompareHashAndPassword([]byte(encodedHash), []byte(password))
		if err != nil {
			return false, nil
		}
		return true, nil
	}

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	var memory, timeParam uint32
	var threads uint8
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeParam, &threads)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	keyLen := uint32(len(decodedHash))

	comparisonHash := argon2.IDKey([]byte(password), salt, timeParam, memory, threads, keyLen)

	if len(decodedHash) != len(comparisonHash) {
		return false, nil
	}

	for i := 0; i < len(decodedHash); i++ {
		if decodedHash[i] != comparisonHash[i] {
			return false, nil
		}
	}

	return true, nil
}
