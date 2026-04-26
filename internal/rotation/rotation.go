package rotation

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/esousa97/gosecretsrotator/internal/crypto"
	"github.com/esousa97/gosecretsrotator/internal/providers/docker"
	"github.com/esousa97/gosecretsrotator/internal/providers/file"
	"github.com/esousa97/gosecretsrotator/internal/storage"
)

// DefaultPasswordLength used when rotating without an explicit length.
const DefaultPasswordLength = 32

// DueSecrets returns the keys of secrets whose IntervalDays has elapsed
// since LastRotated (or that have never been rotated). Secrets with
// IntervalDays <= 0 are skipped (manual-only).
func DueSecrets(store *storage.Store, now time.Time) []string {
	var out []string
	for k, s := range store.Secrets {
		if s == nil || s.IntervalDays <= 0 {
			continue
		}
		interval := time.Duration(s.IntervalDays) * 24 * time.Hour
		if s.LastRotated.IsZero() || now.Sub(s.LastRotated) >= interval {
			out = append(out, k)
		}
	}
	return out
}

// RotateSecret generates a new value for the given secret, applies it to all
// configured targets, then persists the store. Targets are applied in order;
// on the first failure the in-memory value is left updated but Save is not
// called, leaving the on-disk vault unchanged.
func RotateSecret(store *storage.Store, key string) error {
	sec, ok := store.Secrets[key]
	if !ok || sec == nil {
		return fmt.Errorf("secret '%s' not found", key)
	}

	newVal, err := crypto.GeneratePassword(DefaultPasswordLength)
	if err != nil {
		return fmt.Errorf("generate password: %w", err)
	}

	for _, t := range sec.Targets {
		if err := ApplyTarget(t, newVal); err != nil {
			return fmt.Errorf("apply target %+v: %w", t, err)
		}
	}

	sec.Value = newVal
	sec.LastRotated = time.Now().UTC()
	return store.Save()
}

// ApplyTarget dispatches a value to the appropriate provider.
func ApplyTarget(t storage.Target, value string) error {
	switch t.Type {
	case "docker":
		if t.Container == "" || t.EnvKey == "" {
			return fmt.Errorf("docker target requires container and env_key")
		}
		return docker.UpdateContainerEnv(t.Container, t.EnvKey, value)
	case "file":
		if t.Path == "" || t.FileKey == "" {
			return fmt.Errorf("file target requires path and file_key")
		}
		ext := strings.ToLower(filepath.Ext(t.Path))
		switch ext {
		case ".env":
			return file.InjectEnv(t.Path, t.FileKey, value)
		case ".yaml", ".yml":
			return file.InjectYAML(t.Path, t.FileKey, value)
		default:
			return fmt.Errorf("unsupported file extension: %s", ext)
		}
	default:
		return fmt.Errorf("unknown target type: %q", t.Type)
	}
}
