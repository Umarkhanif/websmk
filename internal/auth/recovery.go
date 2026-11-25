package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
)

const recoveryCodeCount = 10

// GenerateRecoveryCodes returns human-readable codes in XXXX-XXXX form. Only
// the hashes are persisted, so this is the one time the codes are visible.
func GenerateRecoveryCodes() ([]string, error) {
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	codes := make([]string, 0, recoveryCodeCount)
	for range recoveryCodeCount {
		buf := make([]byte, 5)
		if _, err := rand.Read(buf); err != nil {
			return nil, fmt.Errorf("generate recovery code: %w", err)
		}
		raw := encoder.EncodeToString(buf)
		codes = append(codes, raw[:4]+"-"+raw[4:])
	}
	return codes, nil
}

// NormalizeRecoveryCode strips spaces and hyphens and uppercases the value.
func NormalizeRecoveryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}

// RecoveryCodeHash binds a code to a user so hashes cannot be replayed across
// accounts. SHA-256 is enough here because codes are random and high entropy.
func RecoveryCodeHash(userID, code string) string {
	sum := sha256.Sum256([]byte("wesmk:recovery:" + userID + ":" + NormalizeRecoveryCode(code)))
	return hex.EncodeToString(sum[:])
}
