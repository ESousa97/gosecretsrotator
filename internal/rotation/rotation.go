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
// configured targets, then persists the store.
func RotateSecret(store *storage.Store, hdb *storage.HistoryDB, key string) error {
	sec, ok := store.Secrets[key]
	if !ok || sec == nil {
		return fmt.Errorf("secret '%s' not found", key)
	}

	// Record initial state if history is empty for this secret
	// This ensures GetLastSuccessful(key, offset 1) works on the first rotation.
	if _, err := hdb.GetLastSuccessful(key); err != nil {
		hdb.Record(key, sec.Value, "success", "initial baseline", "baseline")
	}

	newVal, err := crypto.GeneratePassword(DefaultPasswordLength)
	if err != nil {
		hdb.Record(key, "", "failure", err.Error(), "rotation")
		return fmt.Errorf("generate password: %w", err)
	}

	var applyErr error
	for _, t := range sec.Targets {
		if err := ApplyTarget(t, newVal); err != nil {
			applyErr = fmt.Errorf("apply target %+v: %w", t, err)
			break
		}
	}

	if applyErr != nil {
		hdb.Record(key, newVal, "failure", applyErr.Error(), "rotation")
		return applyErr
	}

	sec.Value = newVal
	sec.LastRotated = time.Now().UTC()
	if err := store.Save(); err != nil {
		hdb.Record(key, newVal, "failure", "save store: "+err.Error(), "rotation")
		return err
	}

	// Archive the OLD value if this is the first successful rotation of this value
	// Or we can record the NEW value as the current state.
	// Requirement: "Each time a secret is rotated, the previous version must be moved to 'history'".
	// Let's record the NEW successful state. The "previous" is naturally the one before the latest in DB.
	hdb.Record(key, newVal, "success", "", "rotation")

	// Ensure we also have a record of the initial state if it's the first time
	// But usually, we just log every successful change.

	return nil
}

// RollbackSecret retrieves the previous successful value and applies it.
func RollbackSecret(store *storage.Store, hdb *storage.HistoryDB, key string) error {
	sec, ok := store.Secrets[key]
	if !ok || sec == nil {
		return fmt.Errorf("secret '%s' not found", key)
	}

	prev, err := hdb.GetLastSuccessful(key)
	if err != nil {
		return fmt.Errorf("get history: %w", err)
	}

	var applyErr error
	for _, t := range sec.Targets {
		if err := ApplyTarget(t, prev.Value); err != nil {
			applyErr = fmt.Errorf("apply target %+v: %w", t, err)
			break
		}
	}

	if applyErr != nil {
		hdb.Record(key, prev.Value, "failure", "rollback: "+applyErr.Error(), "rollback")
		return applyErr
	}

	sec.Value = prev.Value
	sec.LastRotated = time.Now().UTC()
	if err := store.Save(); err != nil {
		hdb.Record(key, prev.Value, "failure", "rollback save: "+err.Error(), "rollback")
		return err
	}

	hdb.Record(key, prev.Value, "success", "", "rollback")
	return nil
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
