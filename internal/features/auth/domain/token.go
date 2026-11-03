package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
)

func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func GenerateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))

	return hex.EncodeToString(h[:])
}
