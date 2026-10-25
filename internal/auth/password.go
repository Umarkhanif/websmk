package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// OWASP Argon2id baseline (19 MiB, t=2, p=1): strong enough for a school VPS
// where many logins can happen at once, without 64 MiB per concurrent login.
const (
	argonMemory      = 19 * 1024
	argonIterations  = 2
	argonParallelism = 1
	argonSaltLength  = 16
	argonKeyLength   = 32
)

var errInvalidHash = errors.New("invalid password hash")

func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonIterations,
		argonParallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(password, encoded string) bool {
	params, salt, expected, err := decodeHash(encoded)
	if err != nil {
		return false
	}

	actual := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func decodeHash(encoded string) (params argonParams, salt, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return params, nil, nil, errInvalidHash
	}

	if _, err = fmt.Sscanf(parts[2], "v=%d", new(int)); err != nil {
		return params, nil, nil, errInvalidHash
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return params, nil, nil, errInvalidHash
	}

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return params, nil, nil, errInvalidHash
	}
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hash) == 0 {
		return params, nil, nil, errInvalidHash
	}

	params = argonParams{memory: memory, iterations: iterations, parallelism: parallelism}
	return params, salt, hash, nil
}
