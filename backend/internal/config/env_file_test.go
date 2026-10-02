package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write test env file: %v", err)
	}
	return path
}

func TestLoadEnvFileParsesValues(t *testing.T) {
	path := writeEnvFile(t, `
# A comment, and the blank line above it, are both ignored.
BDIC_TEST_PLAIN=hello
BDIC_TEST_QUOTED="with spaces"
BDIC_TEST_SINGLE='single quoted'
export BDIC_TEST_EXPORTED=exported
BDIC_TEST_EMPTY=
BDIC_TEST_URL=postgres://user@localhost:5432/bdic?sslmode=disable
`)

	// A password with a dollar sign is more likely than one that wants
	// expanding, so values are taken literally.
	keys := []string{
		"BDIC_TEST_PLAIN", "BDIC_TEST_QUOTED", "BDIC_TEST_SINGLE",
		"BDIC_TEST_EXPORTED", "BDIC_TEST_EMPTY", "BDIC_TEST_URL",
	}
	for _, key := range keys {
		t.Cleanup(func() { _ = os.Unsetenv(key) })
	}

	if err := LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile returned an error: %v", err)
	}

	expected := map[string]string{
		"BDIC_TEST_PLAIN":    "hello",
		"BDIC_TEST_QUOTED":   "with spaces",
		"BDIC_TEST_SINGLE":   "single quoted",
		"BDIC_TEST_EXPORTED": "exported",
		"BDIC_TEST_EMPTY":    "",
		"BDIC_TEST_URL":      "postgres://user@localhost:5432/bdic?sslmode=disable",
	}
	for key, want := range expected {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

// A value already in the environment must win, so what the deployment injects
// is never quietly overridden by a stale file.
func TestLoadEnvFileDoesNotOverrideExistingValues(t *testing.T) {
	const key = "BDIC_TEST_PRECEDENCE"
	t.Setenv(key, "from-the-shell")

	path := writeEnvFile(t, key+"=from-the-file\n")

	if err := LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile returned an error: %v", err)
	}
	if got := os.Getenv(key); got != "from-the-shell" {
		t.Errorf("%s = %q, want the shell value to win", key, got)
	}
}

// Production has no .env at all; it supplies configuration through the
// environment. A missing file must therefore not be an error.
func TestLoadEnvFileIgnoresMissingFile(t *testing.T) {
	if err := LoadEnvFile(filepath.Join(t.TempDir(), "nothing-here")); err != nil {
		t.Errorf("LoadEnvFile on a missing file returned an error: %v", err)
	}
}

func TestLoadEnvFileRejectsMalformedLine(t *testing.T) {
	path := writeEnvFile(t, "THIS_LINE_HAS_NO_EQUALS_SIGN\n")

	if err := LoadEnvFile(path); err == nil {
		t.Error("LoadEnvFile accepted a line with no '=' in it")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("AUTH_SIGNING_KEY", "a-signing-key-that-is-long-enough-yes")

	if _, err := Load(); err == nil {
		t.Error("Load succeeded with no DATABASE_URL")
	}
}

func TestLoadRequiresLongSigningKey(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/bdic")
	t.Setenv("AUTH_SIGNING_KEY", "too-short")

	if _, err := Load(); err == nil {
		t.Error("Load succeeded with a signing key shorter than 32 characters")
	}
}

func TestLoadSucceedsWithTheMinimum(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:5432/bdic")
	t.Setenv("AUTH_SIGNING_KEY", "a-signing-key-that-is-long-enough-yes")
	t.Setenv("APP_ENV", "development")
	t.Setenv("CORS_ORIGINS", "http://localhost:3000, http://127.0.0.1:3000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned an error: %v", err)
	}

	if len(cfg.CORSOrigins) != 2 {
		t.Errorf("CORSOrigins = %v, want two entries", cfg.CORSOrigins)
	}
	if cfg.CORSOrigins[1] != "http://127.0.0.1:3000" {
		t.Errorf("CORSOrigins[1] = %q, want the surrounding space trimmed", cfg.CORSOrigins[1])
	}
	if cfg.IsProduction() {
		t.Error("IsProduction() is true for APP_ENV=development")
	}
	if cfg.TokenTTL <= 0 {
		t.Errorf("TokenTTL = %v, want a positive duration", cfg.TokenTTL)
	}
}

// Production must refuse the configuration that is fine locally and dangerous
// once parents are using it.
func TestLoadRefusesUnsafeProductionConfig(t *testing.T) {
	base := func(t *testing.T) {
		t.Helper()
		t.Setenv("APP_ENV", "production")
		t.Setenv("DATABASE_URL", "postgres://localhost:5432/bdic")
		t.Setenv("AUTH_SIGNING_KEY", "a-signing-key-that-is-long-enough-yes")
		t.Setenv("PUBLIC_BASE_URL", "https://bdic.example")
		t.Setenv("CORS_ORIGINS", "https://bdic.example")
		t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "")
	}

	t.Run("bootstrap password still set", func(t *testing.T) {
		base(t)
		t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "still-here")
		if _, err := Load(); err == nil {
			t.Error("Load allowed a bootstrap admin password in production")
		}
	})

	t.Run("localhost in CORS origins", func(t *testing.T) {
		base(t)
		t.Setenv("CORS_ORIGINS", "http://localhost:3000")
		if _, err := Load(); err == nil {
			t.Error("Load allowed a localhost CORS origin in production")
		}
	})

	t.Run("public base url not https", func(t *testing.T) {
		base(t)
		t.Setenv("PUBLIC_BASE_URL", "http://bdic.example")
		if _, err := Load(); err == nil {
			t.Error("Load allowed a plain http public base URL in production")
		}
	})

	t.Run("a safe production config is accepted", func(t *testing.T) {
		base(t)
		if _, err := Load(); err != nil {
			t.Errorf("Load rejected a safe production config: %v", err)
		}
	})
}
