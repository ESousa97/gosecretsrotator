package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

const (
	masterPwd = "test-master-password"
	vaultFile = "secrets.json"
)

func TestCLI_FullFlow(t *testing.T) {
	// Setup: Remove existing vault if any
	os.Remove(vaultFile)
	defer os.Remove(vaultFile)

	// 1. Add a secret
	secretKey := "DB_PASS"
	secretVal := "super-secure-rotated-val"
	
	cmdAdd := exec.Command("go", "run", "../main.go", "add", secretKey, secretVal)
	cmdAdd.Env = append(os.Environ(), "GOSECRETS_MASTER_PWD="+masterPwd)
	if err := cmdAdd.Run(); err != nil {
		t.Fatalf("Failed to add secret: %v", err)
	}

	// 2. Test Inject into .env
	envFile := "test_output.env"
	origEnv := "DB_PASS=old # original comment"
	os.WriteFile(envFile, []byte(origEnv), 0644)
	defer os.Remove(envFile)

	cmdInjectEnv := exec.Command("go", "run", "../main.go", "inject", "file", "--path", envFile, "--key", "DB_PASS", "--secret", secretKey)
	cmdInjectEnv.Env = append(os.Environ(), "GOSECRETS_MASTER_PWD="+masterPwd)
	if err := cmdInjectEnv.Run(); err != nil {
		t.Fatalf("Failed to inject into .env: %v", err)
	}

	contentEnv, _ := os.ReadFile(envFile)
	if !strings.Contains(string(contentEnv), "DB_PASS="+secretVal+" # original comment") {
		t.Errorf("Expected rotated secret in .env, got: %s", string(contentEnv))
	}

	// 3. Test Inject into .yaml
	yamlFile := "test_output.yaml"
	origYaml := "db:\n  password: old # yaml comment"
	os.WriteFile(yamlFile, []byte(origYaml), 0644)
	defer os.Remove(yamlFile)

	cmdInjectYaml := exec.Command("go", "run", "../main.go", "inject", "file", "--path", yamlFile, "--key", "password", "--secret", secretKey)
	cmdInjectYaml.Env = append(os.Environ(), "GOSECRETS_MASTER_PWD="+masterPwd)
	if err := cmdInjectYaml.Run(); err != nil {
		t.Fatalf("Failed to inject into .yaml: %v", err)
	}

	contentYaml, _ := os.ReadFile(yamlFile)
	if !strings.Contains(string(contentYaml), "password: "+secretVal) {
		t.Errorf("Expected rotated secret in .yaml, got: %s", string(contentYaml))
	}
	if !strings.Contains(string(contentYaml), "# yaml comment") {
		t.Errorf("Expected YAML comment to be preserved, got: %s", string(contentYaml))
	}
}
