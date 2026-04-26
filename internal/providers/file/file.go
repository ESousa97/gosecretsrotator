package file

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// InjectEnv updates a key in a .env file preserving comments and formatting
func InjectEnv(filePath, key, value string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	found := false
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		
		// Check if line starts with KEY=
		if strings.HasPrefix(trimmed, key+"=") {
			// Find if there is a comment in the same line
			parts := strings.SplitN(line, "#", 2)
			newLine := fmt.Sprintf("%s=%s", key, value)
			if len(parts) > 1 {
				newLine += " #" + parts[1]
			}
			lines = append(lines, newLine)
			found = true
		} else {
			lines = append(lines, line)
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

// InjectYAML updates a key in a YAML file using AST to preserve comments
func InjectYAML(filePath, key, value string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return err
	}

	if node.Kind != yaml.DocumentNode {
		return fmt.Errorf("invalid YAML document")
	}

	updated := updateNode(node.Content[0], key, value)
	if !updated {
		// If not found, we'd need more complex logic to add it to AST properly
		// For now, let's assume it exists as per requirements "localizar uma chave específica"
		return fmt.Errorf("key '%s' not found in YAML", key)
	}

	out, err := yaml.Marshal(&node)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, out, 0644)
}

func updateNode(node *yaml.Node, key, value string) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}

	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		if keyNode.Value == key {
			valueNode := node.Content[i+1]
			valueNode.Value = value
			return true
		}
		// Recursive check for nested maps
		if updateNode(node.Content[i+1], key, value) {
			return true
		}
	}
	return false
}
