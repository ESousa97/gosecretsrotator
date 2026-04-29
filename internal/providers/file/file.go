package file

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// splitEnvComment splits a .env line into its content and trailing comment.
// A '#' only starts a comment when it is unquoted and preceded by whitespace
// (or starts the line). This avoids treating '#' inside a value as a comment.
func splitEnvComment(line string) (content, comment string) {
	inSingle, inDouble := false, false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
		case c == '"' && !inSingle:
			inDouble = !inDouble
		case c == '#' && !inSingle && !inDouble:
			if i == 0 || line[i-1] == ' ' || line[i-1] == '\t' {
				return strings.TrimRight(line[:i], " \t"), line[i:]
			}
		}
	}
	return line, ""
}

// InjectEnv updates a key in a .env file preserving comments and formatting
func InjectEnv(filePath, key, value string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	var lines []string
	found := false
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Check if line starts with KEY=
		if strings.HasPrefix(trimmed, key+"=") {
			_, comment := splitEnvComment(line)
			newLine := fmt.Sprintf("%s=%s", key, value)
			if comment != "" {
				newLine += " " + comment
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

	content := []byte(strings.Join(lines, "\n") + "\n")
	return os.WriteFile(filePath, content, 0600)
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

	return os.WriteFile(filePath, out, 0600)
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
