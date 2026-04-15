package storage

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/esousa97/gosecretsrotator/internal/crypto"
)

// Store represents the collection of secrets
type Store struct {
	filePath string
	password string
	Secrets  map[string]string `json:"secrets"`
}

// NewStore initializes a storage handler
func NewStore(filePath, password string) *Store {
	return &Store{
		filePath: filePath,
		password: password,
		Secrets:  make(map[string]string),
	}
}

// Load decrypts and reads secrets from the JSON file
func (s *Store) Load() error {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return nil // File does not exist, start with empty store
	}

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	plaintext, err := crypto.Decrypt(data, s.password)
	if err != nil {
		return fmt.Errorf("failed to decrypt storage: ensure master password is correct")
	}

	return json.Unmarshal(plaintext, &s.Secrets)
}

// Save encrypts and writes secrets to the JSON file
func (s *Store) Save() error {
	plaintext, err := json.Marshal(s.Secrets)
	if err != nil {
		return err
	}

	ciphertext, err := crypto.Encrypt(plaintext, s.password)
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, ciphertext, 0600) // Restricted permissions
}
