package request

import (
	"os"
	"path/filepath"
	"testing"
)

// TestReadNodeSiteConfigWithTOML verifies that ReadNodeSiteConfig
// correctly loads site configuration from config.toml
func TestReadNodeSiteConfigWithTOML(t *testing.T) {
	withIsolatedGlobals(t)

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")
	configContent := `
[sites."example.com"]
https_agent = true

[sites."example.com".headers]
X-Custom = "test-value"

[sites."test.org"]
https_agent = false
`
	if err := os.WriteFile(configFile, []byte(configContent), 0o600); err != nil {
		t.Fatalf("failed to create config.toml: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	config := ReadNodeSiteConfig()
	
	// Verify example.com configuration
	exampleConfig, ok := config["example.com"]
	if !ok {
		t.Fatalf("expected example.com config to exist")
	}
	if exampleConfig.HttpsAgent != "true" {
		t.Fatalf("expected httpsAgent to be 'true', got %q", exampleConfig.HttpsAgent)
	}
	if exampleConfig.Headers["X-Custom"] != "test-value" {
		t.Fatalf("expected X-Custom header to be 'test-value', got %q", exampleConfig.Headers["X-Custom"])
	}
	
	// Verify test.org configuration
	testConfig, ok := config["test.org"]
	if !ok {
		t.Fatalf("expected test.org config to exist")
	}
	// When https_agent is false in TOML, it becomes empty string in the config
	if testConfig.HttpsAgent != "" {
		t.Fatalf("expected httpsAgent to be empty string (false), got %q", testConfig.HttpsAgent)
	}
}

// TestReadNodeSiteConfigTOMLPriorityOverLegacy verifies that config.toml
// takes priority over node-site-config.json
func TestReadNodeSiteConfigTOMLPriorityOverLegacy(t *testing.T) {
	withIsolatedGlobals(t)

	tempDir := t.TempDir()
	
	// Create legacy config
	legacyFile := filepath.Join(tempDir, "node-site-config.json")
	legacyContent := `{"legacy.com":{"httpsAgent":"yes","headers":{"X-Legacy":"old"}}}`
	if err := os.WriteFile(legacyFile, []byte(legacyContent), 0o600); err != nil {
		t.Fatalf("failed to create legacy config: %v", err)
	}
	
	// Create TOML config
	tomlFile := filepath.Join(tempDir, "config.toml")
	tomlContent := `
[sites."toml.com"]
https_agent = true

[sites."toml.com".headers]
X-TOML = "new"
`
	if err := os.WriteFile(tomlFile, []byte(tomlContent), 0o600); err != nil {
		t.Fatalf("failed to create config.toml: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	config := ReadNodeSiteConfig()
	
	// TOML config should be loaded
	tomlConfig, ok := config["toml.com"]
	if !ok {
		t.Fatalf("expected toml.com config to exist")
	}
	if tomlConfig.Headers["X-TOML"] != "new" {
		t.Fatalf("expected X-TOML header to be 'new', got %q", tomlConfig.Headers["X-TOML"])
	}
	
	// Legacy config should NOT be loaded when TOML exists
	if _, ok := config["legacy.com"]; ok {
		t.Fatalf("expected legacy.com config to NOT exist when TOML is present")
	}
}
