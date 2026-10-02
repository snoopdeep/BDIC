package config

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// LoadEnvFile reads a .env file into the process environment.
//
// Written out rather than pulled from a dependency, because the whole job is
// forty lines and a dependency for it is a dependency to keep updated forever.
//
// A variable already present in the environment is never overwritten, so a
// value exported in the shell, or injected by the deployment, always beats the
// file. A missing file is not an error: production supplies its configuration
// through the environment and has no .env at all.
func LoadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// `export FOO=bar` is common in files people also source in a shell.
		line = strings.TrimPrefix(line, "export ")

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("%s line %d: expected KEY=value", path, lineNumber)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("%s line %d: empty variable name", path, lineNumber)
		}

		value = strings.TrimSpace(value)
		value = unquote(value)

		// Already set wins.
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("%s line %d: set %s: %w", path, lineNumber, key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

// unquote strips one matching pair of surrounding quotes. Values are otherwise
// taken literally: no variable expansion, because a password containing a
// dollar sign is more likely than a password that wants expanding.
func unquote(value string) string {
	if len(value) < 2 {
		return value
	}
	first, last := value[0], value[len(value)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
		return value[1 : len(value)-1]
	}
	return value
}
