package config

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear flags from previous tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Clear environment variables
	os.Unsetenv("PORT")
	os.Unsetenv("TVCLIPBOARD_SESSION_TIMEOUT")
	os.Unsetenv("TVCLIPBOARD_PRIVATE_KEY")

	cfg := Load()

	if cfg.Port != "3333" {
		t.Errorf("Expected default port 3333, got %s", cfg.Port)
	}
	if cfg.SessionTimeout != 10*time.Minute {
		t.Errorf("Expected default timeout 10m, got %v", cfg.SessionTimeout)
	}
	if cfg.PrivateKeyHex != "" {
		t.Errorf("Expected empty private key, got %s", cfg.PrivateKeyHex)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Clear flags from previous tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Setenv("PORT", "8080")
	os.Setenv("TVCLIPBOARD_SESSION_TIMEOUT", "15")
	os.Setenv("TVCLIPBOARD_PRIVATE_KEY", "abcdef123456")

	defer os.Unsetenv("PORT")
	defer os.Unsetenv("TVCLIPBOARD_SESSION_TIMEOUT")
	defer os.Unsetenv("TVCLIPBOARD_PRIVATE_KEY")

	cfg := Load()

	if cfg.Port != "8080" {
		t.Errorf("Expected port 8080 from env, got %s", cfg.Port)
	}
	if cfg.SessionTimeout != 15*time.Minute {
		t.Errorf("Expected timeout 15m from env, got %v", cfg.SessionTimeout)
	}
	if cfg.PrivateKeyHex != "abcdef123456" {
		t.Errorf("Expected private key from env, got %s", cfg.PrivateKeyHex)
	}
}

func TestLoadFromCLI(t *testing.T) {
	// Clear flags from previous tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Simulate CLI arguments
	oldArgs := os.Args
	os.Args = []string{"tvclipboard", "--port", "9000", "--expires", "20", "--key", "deadbeef"}
	defer func() { os.Args = oldArgs }()

	cfg := Load()

	if cfg.Port != "9000" {
		t.Errorf("Expected port 9000 from CLI, got %s", cfg.Port)
	}
	if cfg.SessionTimeout != 20*time.Minute {
		t.Errorf("Expected timeout 20m from CLI, got %v", cfg.SessionTimeout)
	}
	if cfg.PrivateKeyHex != "deadbeef" {
		t.Errorf("Expected private key from CLI, got %s", cfg.PrivateKeyHex)
	}
}

func TestLoadCLIOverridesEnv(t *testing.T) {
	// Clear flags from previous tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Setenv("PORT", "8080")
	os.Setenv("TVCLIPBOARD_SESSION_TIMEOUT", "15")

	defer os.Unsetenv("PORT")
	defer os.Unsetenv("TVCLIPBOARD_SESSION_TIMEOUT")

	// Simulate CLI arguments
	oldArgs := os.Args
	os.Args = []string{"tvclipboard", "--port", "9000", "--expires", "20"}
	defer func() { os.Args = oldArgs }()

	cfg := Load()

	// CLI should override ENV
	if cfg.Port != "9000" {
		t.Errorf("Expected CLI port 9000 to override ENV 8080, got %s", cfg.Port)
	}
	if cfg.SessionTimeout != 20*time.Minute {
		t.Errorf("Expected CLI timeout 20m to override ENV 15m, got %v", cfg.SessionTimeout)
	}
}

func TestLoadInvalidEnvTimeout(t *testing.T) {
	// Clear flags from previous tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Setenv("TVCLIPBOARD_SESSION_TIMEOUT", "invalid")
	defer os.Unsetenv("TVCLIPBOARD_SESSION_TIMEOUT")

	cfg := Load()

	// Should fall back to default
	if cfg.SessionTimeout != 10*time.Minute {
		t.Errorf("Expected default timeout 10m for invalid env, got %v", cfg.SessionTimeout)
	}
}

func TestLoadZeroTimeout(t *testing.T) {
	// Clear flags from previous tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Setenv("TVCLIPBOARD_SESSION_TIMEOUT", "0")
	defer os.Unsetenv("TVCLIPBOARD_SESSION_TIMEOUT")

	cfg := Load()

	// Should fall back to default
	if cfg.SessionTimeout != 10*time.Minute {
		t.Errorf("Expected default timeout 10m for zero env, got %v", cfg.SessionTimeout)
	}
}

func TestLoadNegativeTimeout(t *testing.T) {
	// Clear flags from previous tests
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	os.Setenv("TVCLIPBOARD_SESSION_TIMEOUT", "-5")
	defer os.Unsetenv("TVCLIPBOARD_SESSION_TIMEOUT")

	cfg := Load()

	// Should fall back to default
	if cfg.SessionTimeout != 10*time.Minute {
		t.Errorf("Expected default timeout 10m for negative env, got %v", cfg.SessionTimeout)
	}
}

func TestGetQRHostDefault(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Unsetenv("TVCLIPBOARD_PUBLIC_URL")

	cfg := Load()

	expectedHost := cfg.LocalIP
	if cfg.GetQRHost() != expectedHost {
		t.Errorf("Expected GetQRHost to return local IP %s, got %s", expectedHost, cfg.GetQRHost())
	}
}

func TestGetQRHostPublicURL(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Setenv("TVCLIPBOARD_PUBLIC_URL", "https://example.io")
	defer os.Unsetenv("TVCLIPBOARD_PUBLIC_URL")

	cfg := Load()

	if cfg.GetQRHost() != "example.io" {
		t.Errorf("Expected GetQRHost to return example.io, got %s", cfg.GetQRHost())
	}
}

func TestGetQRHostPublicURLWithPort(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Setenv("TVCLIPBOARD_PUBLIC_URL", "https://example.io:3333")
	defer os.Unsetenv("TVCLIPBOARD_PUBLIC_URL")

	cfg := Load()

	if cfg.GetQRHost() != "example.io" {
		t.Errorf("Expected GetQRHost to return example.io, got %s", cfg.GetQRHost())
	}
}

func TestGetQRSchemeDefault(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Unsetenv("TVCLIPBOARD_PUBLIC_URL")

	cfg := Load()

	if cfg.GetQRScheme() != "http" {
		t.Errorf("Expected GetQRScheme to return http, got %s", cfg.GetQRScheme())
	}
}

func TestGetQRSchemePublicURL(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Setenv("TVCLIPBOARD_PUBLIC_URL", "https://example.io")
	defer os.Unsetenv("TVCLIPBOARD_PUBLIC_URL")

	cfg := Load()

	if cfg.GetQRScheme() != "https" {
		t.Errorf("Expected GetQRScheme to return https, got %s", cfg.GetQRScheme())
	}
}

func TestGetQRSchemePublicURLWithoutScheme(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Setenv("TVCLIPBOARD_PUBLIC_URL", "example.io")
	defer os.Unsetenv("TVCLIPBOARD_PUBLIC_URL")

	cfg := Load()

	if cfg.GetQRScheme() != "http" {
		t.Errorf("Expected GetQRScheme to return http, got %s", cfg.GetQRScheme())
	}
}

func TestPublicURLFromCLI(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Unsetenv("TVCLIPBOARD_PUBLIC_URL")

	oldArgs := os.Args
	os.Args = []string{"tvclipboard", "--base-url", "https://example.com"}
	defer func() { os.Args = oldArgs }()

	cfg := Load()

	if cfg.PublicURL != "https://example.com" {
		t.Errorf("Expected PublicURL from CLI, got %s", cfg.PublicURL)
	}
}
