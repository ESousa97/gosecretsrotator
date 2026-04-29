package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type HistoryEntry struct {
	ID         int
	SecretName string
	Value      string
	RotatedAt  time.Time
	Status     string // "success", "failure"
	ErrorMsg   string
	Operation  string // "rotation", "rollback"
}

type HistoryDB struct {
	db *sql.DB
}

func NewHistoryDB(path string) (*HistoryDB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open history db: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS secret_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		secret_name TEXT NOT NULL,
		value TEXT NOT NULL,
		rotated_at DATETIME NOT NULL,
		status TEXT NOT NULL,
		error_msg TEXT,
		operation TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_secret_name ON secret_history(secret_name);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &HistoryDB{db: db}, nil
}

func (h *HistoryDB) Close() error {
	return h.db.Close()
}

func (h *HistoryDB) Record(name, value, status, errMsg, operation string) error {
	_, err := h.db.Exec(
		"INSERT INTO secret_history (secret_name, value, rotated_at, status, error_msg, operation) VALUES (?, ?, ?, ?, ?, ?)",
		name, value, time.Now().UTC(), status, errMsg, operation,
	)
	if err != nil {
		log.Printf("failed to record history: %v", err)
	}
	return err
}

func (h *HistoryDB) GetLastSuccessful(name string) (*HistoryEntry, error) {
	row := h.db.QueryRow(
		"SELECT id, secret_name, value, rotated_at, status, error_msg, operation FROM secret_history WHERE secret_name = ? AND status = 'success' ORDER BY rotated_at DESC LIMIT 1 OFFSET 1",
		name,
	)
	var entry HistoryEntry
	err := row.Scan(&entry.ID, &entry.SecretName, &entry.Value, &entry.RotatedAt, &entry.Status, &entry.ErrorMsg, &entry.Operation)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no previous version found for %s", name)
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}
