package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoad_DatabasePath verifies database path configuration
func TestLoad_DatabasePath(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Test 1: Default database path when no config
	cfg, _, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Database.Path != "db.sqlite" {
		t.Errorf("Expected default database path 'db.sqlite', got %q", cfg.Database.Path)
	}

	// Test 2: TOML with absolute database path
	absPath := filepath.Join(tmpDir, "test.db")
	tomlContent := `
[database]
path = "` + filepath.ToSlash(absPath) + `"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write config.toml: %v", err)
	}

	cfg, _, err = Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	// On Windows, the path might be normalized differently
	if filepath.Clean(cfg.Database.Path) != filepath.Clean(absPath) {
		t.Errorf("Expected database path %q, got %q", absPath, cfg.Database.Path)
	}

	// Test 3: TOML with relative database path
	tomlContent = `
[database]
path = "data/cache.db"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write config.toml: %v", err)
	}

	// Debug: Check if config file can be found
	foundPath, found := FindConfigFile()
	if !found {
		t.Fatalf("config.toml not found in %s", tmpDir)
	}
	t.Logf("Found config.toml at: %s", foundPath)

	cfg, _, err = Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	expectedPath := filepath.Join(tmpDir, "data", "cache.db")
	t.Logf("Expected: %s, Got: %s", expectedPath, cfg.Database.Path)
	if cfg.Database.Path != expectedPath {
		t.Errorf("Expected database path %q, got %q", expectedPath, cfg.Database.Path)
	}
}

// TestResolveDatabasePath verifies database path resolution
func TestResolveDatabasePath(t *testing.T) {
	tests := []struct {
		name     string
		dbPath   string
		tomlPath string
		expected string
	}{
		{
			name:     "Empty path returns empty",
			dbPath:   "",
			tomlPath: "/home/user/config.toml",
			expected: "",
		},
		{
			name:     "Relative path resolved with Unix separator",
			dbPath:   "data/cache.db",
			tomlPath: "/home/user/.config/rss2cloud/config.toml",
			expected: "/home/user/.config/rss2cloud/data/cache.db",
		},
		{
			name:     "Relative path with dot",
			dbPath:   "./db.sqlite",
			tomlPath: "/home/user/config.toml",
			expected: "/home/user/db.sqlite",
		},
		{
			name:     "No toml path returns relative as-is",
			dbPath:   "data/cache.db",
			tomlPath: "",
			expected: "data/cache.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveDatabasePath(tt.dbPath, tt.tomlPath)
			// Normalize paths for comparison
			gotNorm := filepath.ToSlash(got)
			expectedNorm := filepath.ToSlash(tt.expected)
			if gotNorm != expectedNorm {
				t.Errorf("ResolveDatabasePath(%q, %q) = %q, want %q", tt.dbPath, tt.tomlPath, gotNorm, expectedNorm)
			}
		})
	}

	// Test absolute path separately (platform-specific)
	t.Run("Absolute path unchanged", func(t *testing.T) {
		absPath := filepath.Join(os.TempDir(), "test.db")
		tomlPath := filepath.Join(os.TempDir(), "config.toml")
		got := ResolveDatabasePath(absPath, tomlPath)
		if got != absPath {
			t.Errorf("ResolveDatabasePath(%q, %q) = %q, want %q", absPath, tomlPath, got, absPath)
		}
	})
}

// TestLoad_DatabasePathInUserConfigDir verifies database path resolution in user config directory
func TestLoad_DatabasePathInUserConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "rss2cloud")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Set HOME to tmpDir
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE") // Windows uses USERPROFILE
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	// Create config.toml in user config directory
	tomlContent := `
[database]
path = "app.db"
`
	configPath := filepath.Join(configDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write config.toml: %v", err)
	}

	// Change to a different working directory
	workDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(workDir)

	// Debug: Check if config file can be found
	foundPath, found := FindConfigFile()
	if !found {
		t.Fatalf("config.toml not found. Expected at: %s", configPath)
	}
	t.Logf("Found config.toml at: %s", foundPath)

	cfg, _, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Database path should be resolved relative to config.toml location
	expectedPath := filepath.Join(configDir, "app.db")
	t.Logf("Expected: %s, Got: %s", expectedPath, cfg.Database.Path)
	if cfg.Database.Path != expectedPath {
		t.Errorf("Expected database path %q, got %q", expectedPath, cfg.Database.Path)
	}
}

func TestLoad_DefaultDatabasePathFallsBackToUserConfigDir(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "rss2cloud")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}
	dbPath := filepath.Join(configDir, "db.sqlite")
	if err := os.WriteFile(dbPath, []byte("placeholder"), 0o600); err != nil {
		t.Fatalf("Failed to create user config db.sqlite: %v", err)
	}

	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpDir)
	os.Setenv("USERPROFILE", tmpDir)
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	workDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(workDir)

	cfg, _, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if cfg.Database.Path != dbPath {
		t.Errorf("Expected database path %q, got %q", dbPath, cfg.Database.Path)
	}
}
