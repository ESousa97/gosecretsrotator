package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const passwordAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*-_=+"

// GeneratePassword returns a cryptographically random password of the given
// length drawn from a mixed alphanumeric+symbol alphabet. Uses crypto/rand.
func GeneratePassword(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("password length must be > 0")
	}
	out := make([]byte, length)
	alphabetSize := big.NewInt(int64(len(passwordAlphabet)))
	for i := range out {
		n, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", fmt.Errorf("rand failure: %w", err)
		}
		out[i] = passwordAlphabet[n.Int64()]
	}
	return string(out), nil
}
