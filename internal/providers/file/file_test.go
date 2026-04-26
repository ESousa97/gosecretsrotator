package file

import (
	"os"
	"strings"
	"testing"
)

func TestInjectEnv(t *testing.T) {
	content := "# Config\nDB_PASS=old # pass\nOTHER=val"
	tmpFile := "test_inject.env"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err := InjectEnv(tmpFile, "DB_PASS", "new_secret")
	if err != nil {
		t.Fatalf("Failed to inject env: %v", err)
	}

	result, _ := os.ReadFile(tmpFile)
	resStr := string(result)

	if !strings.Contains(resStr, "DB_PASS=new_secret # pass") {
		t.Errorf("Expected updated value with comment, got: %s", resStr)
	}
	if !strings.Contains(resStr, "# Config") {
		t.Errorf("Expected comment to be preserved, got: %s", resStr)
	}
}

func TestInjectYAML(t *testing.T) {
	content := "db:\n  pass: old # secret\n  user: admin"
	tmpFile := "test_inject.yaml"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile) }()

	err := InjectYAML(tmpFile, "pass", "new_secret")
	if err != nil {
		t.Fatalf("Failed to inject yaml: %v", err)
	}

	result, _ := os.ReadFile(tmpFile)
	resStr := string(result)

	// Note: yaml.v3 might change indentation but should keep comments
	if !strings.Contains(resStr, "pass: new_secret") {
		t.Errorf("Expected updated value, got: %s", resStr)
	}
	if !strings.Contains(resStr, "# secret") {
		t.Errorf("Expected comment to be preserved, got: %s", resStr)
	}
}
