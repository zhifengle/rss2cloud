package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setTestHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
}

func TestReadConfigFileFallsBackToUserConfigDir(t *testing.T) {
	tempDir := t.TempDir()
	setTestHome(t, tempDir)

	configDir := filepath.Join(tempDir, ".config", appDirName)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configFile := filepath.Join(configDir, "rss.json")
	if err := os.WriteFile(configFile, []byte(`{"example.com":[]}`), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	workDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	data, path, err := readConfigFile("rss.json", false)
	if err != nil {
		t.Fatalf("expected config file to be read: %v", err)
	}
	if string(data) != `{"example.com":[]}` {
		t.Fatalf("unexpected config content: %s", data)
	}
	if path != configFile {
		t.Fatalf("expected path %q, got %q", configFile, path)
	}
}

func TestReadConfigFilePrefersWorkingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	setTestHome(t, tempDir)

	configDir := filepath.Join(tempDir, ".config", appDirName)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "rss.json"), []byte("from-config"), 0o600); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "rss.json"), []byte("from-workdir"), 0o600); err != nil {
		t.Fatalf("failed to write workdir file: %v", err)
	}
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	data, path, err := readConfigFile("rss.json", false)
	if err != nil {
		t.Fatalf("expected config file to be read: %v", err)
	}
	if string(data) != "from-workdir" {
		t.Fatalf("unexpected config content: %s", data)
	}
	// Path should now be absolute
	expectedPath, _ := filepath.Abs("rss.json")
	if path != expectedPath {
		t.Fatalf("expected absolute workdir path %q, got %q", expectedPath, path)
	}
}
