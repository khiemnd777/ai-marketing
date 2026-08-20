package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func NewOpaqueToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate secure token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func TokenDigest(secret []byte, token string) []byte {
	hash := hmac.New(sha256.New, secret)
	_, _ = hash.Write([]byte(token))
	return hash.Sum(nil)
}
