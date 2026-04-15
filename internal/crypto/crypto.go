package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	salt          = "gosecretsrotator-salt-2026" // Fixed salt for simplicity in base tool
	keyLength     = 32                         // 32 bytes for AES-256
	pbkdf2Iter    = 10000                      // Standard PBKDF2 iterations
)

// Encrypt takes a plaintext and a master password, returning the encrypted blob
func Encrypt(plaintext []byte, password string) ([]byte, error) {
	key := deriveKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Seal appends the ciphertext to the nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt takes a ciphertext and a master password, returning the plaintext
func Decrypt(ciphertext []byte, password string) ([]byte, error) {
	key := deriveKey(password)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, encryptedData, nil)
}

// deriveKey generates a 32-byte key from the password using PBKDF2
func deriveKey(password string) []byte {
	return pbkdf2.Key([]byte(password), []byte(salt), pbkdf2Iter, keyLength, sha256.New)
}
