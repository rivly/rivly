package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

const setupTokenBytes = 32

func NewSetupToken() (string, error) {
	buf := make([]byte, setupTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate setup token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func SetupTokenMatches(want, got string) bool {
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}
