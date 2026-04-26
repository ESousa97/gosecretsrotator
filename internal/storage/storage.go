package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/esousa97/gosecretsrotator/internal/crypto"
)

// Target describes where a secret value must be applied when rotated.
type Target struct {
	Type      string `json:"type"` // "docker" | "file"
	Container string `json:"container,omitempty"`
	EnvKey    string `json:"env_key,omitempty"`
	Path      string `json:"path,omitempty"`
	FileKey   string `json:"file_key,omitempty"`
}

// Secret holds a value plus rotation metadata and propagation targets.
type Secret struct {
	Value        string    `json:"value"`
	LastRotated  time.Time `json:"last_rotated"`
	IntervalDays int       `json:"interval_days,omitempty"`
	Targets      []Target  `json:"targets,omitempty"`
}

// Store represents the collection of secrets
type Store struct {
	filePath string
	password string
	Secrets  map[string]*Secret `json:"secrets"`
}

// NewStore initializes a storage handler
func NewStore(filePath, password string) *Store {
	return &Store{
		filePath: filePath,
		password: password,
		Secrets:  make(map[string]*Secret),
	}
}

// Load decrypts and reads secrets from the JSON file
func (s *Store) Load() error {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return nil
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

	if err := json.Unmarshal(plaintext, &s.Secrets); err != nil {
		// Backward-compat: old vaults stored map[string]string.
		var legacy map[string]string
		if err2 := json.Unmarshal(plaintext, &legacy); err2 != nil {
			return err
		}
		s.Secrets = make(map[string]*Secret, len(legacy))
		for k, v := range legacy {
			s.Secrets[k] = &Secret{Value: v, LastRotated: time.Now().UTC()}
		}
	}
	return nil
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

	return os.WriteFile(s.filePath, ciphertext, 0600)
}
