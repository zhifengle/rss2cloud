package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoad_TOMLOnly verifies Load() works with TOML configuration only
func TestLoad_TOMLOnly(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a config.toml file
	tomlContent := `
[auth]
cookies = "TEST_COOKIES"

[server]
port = 9000

[p115]
disable_cache = true
chunk_delay = 3
chunk_size = 250

[proxy]
http = "http://proxy.example.com:8080"

[[rss]]
site = "example.com"
name = "test-feed"
url = "http://example.com/rss"

[sites."example.com"]
https_agent = true
`

	tomlPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config.toml: %v", err)
	}

	// Change to temp directory so FindConfigFile finds our test file
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify source tracking
	if source.TOMLPath == "" {
		t.Error("Expected TOMLPath to be set")
	}

	// Verify auth configuration
	if cfg.Auth.Cookies != "TEST_COOKIES" {
		t.Errorf("Expected cookies 'TEST_COOKIES', got %s", cfg.Auth.Cookies)
	}

	// Verify server configuration
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
	}

	// Verify P115 configuration
	if !cfg.P115.DisableCache {
		t.Error("Expected DisableCache true")
	}
	if cfg.P115.ChunkDelay != 3 {
		t.Errorf("Expected ChunkDelay 3, got %d", cfg.P115.ChunkDelay)
	}
	if cfg.P115.ChunkSize != 250 {
		t.Errorf("Expected ChunkSize 250, got %d", cfg.P115.ChunkSize)
	}

	// Verify proxy configuration
	if cfg.Proxy.HTTP != "http://proxy.example.com:8080" {
		t.Errorf("Expected proxy 'http://proxy.example.com:8080', got %s", cfg.Proxy.HTTP)
	}

	// Verify RSS configuration
	if len(cfg.RSS) != 1 {
		t.Errorf("Expected 1 RSS site, got %d", len(cfg.RSS))
	}
	if feeds, ok := cfg.RSS["example.com"]; !ok || len(feeds) != 1 {
		t.Error("Expected RSS feed for example.com")
	} else if feeds[0].Name != "test-feed" {
		t.Errorf("Expected feed name 'test-feed', got %s", feeds[0].Name)
	}

	// Verify sites configuration
	if len(cfg.Sites) != 1 {
		t.Errorf("Expected 1 site, got %d", len(cfg.Sites))
	}
	if site, ok := cfg.Sites["example.com"]; !ok || site.HttpsAgent != "true" {
		t.Error("Expected site example.com with https_agent=true")
	}
}

// TestLoad_LegacyOnly verifies Load() works with legacy configuration files only
func TestLoad_LegacyOnly(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create legacy rss.json
	rssContent := `{
		"example.com": [
			{
				"name": "legacy-feed",
				"url": "http://example.com/legacy-rss"
			}
		]
	}`
	rssPath := filepath.Join(tmpDir, "rss.json")
	if err := os.WriteFile(rssPath, []byte(rssContent), 0644); err != nil {
		t.Fatalf("Failed to write test rss.json: %v", err)
	}

	// Create legacy node-site-config.json
	sitesContent := `{
		"legacy.com": {
			"httpsAgent": "true"
		}
	}`
	sitesPath := filepath.Join(tmpDir, "node-site-config.json")
	if err := os.WriteFile(sitesPath, []byte(sitesContent), 0644); err != nil {
		t.Fatalf("Failed to write test node-site-config.json: %v", err)
	}

	// Create legacy .cookies
	cookiesContent := "LEGACY_COOKIES"
	cookiesPath := filepath.Join(tmpDir, ".cookies")
	if err := os.WriteFile(cookiesPath, []byte(cookiesContent), 0644); err != nil {
		t.Fatalf("Failed to write test .cookies: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify source tracking
	if source.TOMLPath != "" {
		t.Error("Expected TOMLPath to be empty for legacy-only config")
	}
	if source.CookiesPath == "" {
		t.Error("Expected CookiesPath to be set")
	}

	// Verify auth configuration from legacy
	if cfg.Auth.Cookies != "LEGACY_COOKIES" {
		t.Errorf("Expected cookies 'LEGACY_COOKIES', got %s", cfg.Auth.Cookies)
	}

	// Verify RSS configuration from legacy
	if len(cfg.RSS) != 1 {
		t.Errorf("Expected 1 RSS site, got %d", len(cfg.RSS))
	}
	if feeds, ok := cfg.RSS["example.com"]; !ok || len(feeds) != 1 {
		t.Error("Expected RSS feed for example.com")
	} else if feeds[0].Name != "legacy-feed" {
		t.Errorf("Expected feed name 'legacy-feed', got %s", feeds[0].Name)
	}

	// Verify sites configuration from legacy
	if len(cfg.Sites) != 1 {
		t.Errorf("Expected 1 site, got %d", len(cfg.Sites))
		for k, v := range cfg.Sites {
			t.Logf("Site: %s, HttpsAgent: %s", k, v.HttpsAgent)
		}
	}
	if site, ok := cfg.Sites["legacy.com"]; !ok {
		t.Error("Expected site legacy.com to exist")
	} else if site.HttpsAgent != "true" {
		t.Errorf("Expected site legacy.com with https_agent=true, got %s", site.HttpsAgent)
	}

	// Verify defaults are used for other settings
	if cfg.Server.Port != 8115 {
		t.Errorf("Expected default port 8115, got %d", cfg.Server.Port)
	}
	if cfg.Proxy.HTTP != "http://127.0.0.1:10809" {
		t.Errorf("Expected default proxy, got %s", cfg.Proxy.HTTP)
	}
}

// TestLoad_MixedSources verifies Load() works with mixed TOML and legacy sources
func TestLoad_MixedSources(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a config.toml with only auth and server sections
	tomlContent := `
[auth]
cookies = "TOML_COOKIES"

[server]
port = 9000
`
	tomlPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config.toml: %v", err)
	}

	// Create legacy rss.json (should be used since TOML has no RSS)
	rssContent := `{
		"mixed.com": [
			{
				"name": "mixed-feed",
				"url": "http://mixed.com/rss"
			}
		]
	}`
	rssPath := filepath.Join(tmpDir, "rss.json")
	if err := os.WriteFile(rssPath, []byte(rssContent), 0644); err != nil {
		t.Fatalf("Failed to write test rss.json: %v", err)
	}

	// Create legacy node-site-config.json (should be used since TOML has no sites)
	sitesContent := `{
		"mixed.com": {
			"httpsAgent": "true"
		}
	}`
	sitesPath := filepath.Join(tmpDir, "node-site-config.json")
	if err := os.WriteFile(sitesPath, []byte(sitesContent), 0644); err != nil {
		t.Fatalf("Failed to write test node-site-config.json: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify source tracking
	if source.TOMLPath == "" {
		t.Error("Expected TOMLPath to be set")
	}

	// Verify auth from TOML
	if cfg.Auth.Cookies != "TOML_COOKIES" {
		t.Errorf("Expected cookies 'TOML_COOKIES', got %s", cfg.Auth.Cookies)
	}

	// Verify server from TOML
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected port 9000, got %d", cfg.Server.Port)
	}

	// Verify RSS from legacy
	if len(cfg.RSS) != 1 {
		t.Errorf("Expected 1 RSS site, got %d", len(cfg.RSS))
	}
	if feeds, ok := cfg.RSS["mixed.com"]; !ok || len(feeds) != 1 {
		t.Error("Expected RSS feed for mixed.com")
	}

	// Verify sites from legacy
	if len(cfg.Sites) != 1 {
		t.Errorf("Expected 1 site, got %d", len(cfg.Sites))
	}
	if site, ok := cfg.Sites["mixed.com"]; !ok || site.HttpsAgent != "true" {
		t.Error("Expected site mixed.com with https_agent=true")
	}
}

// TestLoad_CLIOverrides verifies CLI parameters override all file-based configuration
func TestLoad_CLIOverrides(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a config.toml
	tomlContent := `
[auth]
cookies = "TOML_COOKIES"

[server]
port = 9000

[p115]
chunk_delay = 3

[[rss]]
site = "toml.com"
name = "toml-feed"
url = "http://toml.com/rss"
`
	tomlPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config.toml: %v", err)
	}

	// Create a CLI RSS file
	cliRSSContent := `{
		"cli.com": [
			{
				"name": "cli-feed",
				"url": "http://cli.com/rss"
			}
		]
	}`
	cliRSSPath := filepath.Join(tmpDir, "cli-rss.json")
	if err := os.WriteFile(cliRSSPath, []byte(cliRSSContent), 0644); err != nil {
		t.Fatalf("Failed to write test cli-rss.json: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration with CLI overrides
	cfg, _, err := Load(CLIParams{
		Cookies:    "CLI_COOKIES",
		Port:       8888,
		ChunkDelay: 10,
		RSSPath:    cliRSSPath,
	})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify CLI overrides TOML for auth
	if cfg.Auth.Cookies != "CLI_COOKIES" {
		t.Errorf("Expected CLI cookies 'CLI_COOKIES', got %s", cfg.Auth.Cookies)
	}

	// Verify CLI overrides TOML for server
	if cfg.Server.Port != 8888 {
		t.Errorf("Expected CLI port 8888, got %d", cfg.Server.Port)
	}

	// Verify CLI overrides TOML for P115
	if cfg.P115.ChunkDelay != 10 {
		t.Errorf("Expected CLI ChunkDelay 10, got %d", cfg.P115.ChunkDelay)
	}

	// Verify CLI RSS path overrides TOML RSS
	if len(cfg.RSS) != 1 {
		t.Errorf("Expected 1 RSS site, got %d", len(cfg.RSS))
	}
	if feeds, ok := cfg.RSS["cli.com"]; !ok || len(feeds) != 1 {
		t.Error("Expected RSS feed for cli.com from CLI")
	} else if feeds[0].Name != "cli-feed" {
		t.Errorf("Expected feed name 'cli-feed', got %s", feeds[0].Name)
	}
}

// TestLoad_CookiesFileResolution verifies cookies_file path resolution
func TestLoad_CookiesFileResolution(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a cookies file in a subdirectory
	cookiesDir := filepath.Join(tmpDir, "auth")
	if err := os.MkdirAll(cookiesDir, 0755); err != nil {
		t.Fatalf("Failed to create auth directory: %v", err)
	}
	cookiesContent := "FILE_COOKIES"
	cookiesPath := filepath.Join(cookiesDir, "my.cookies")
	if err := os.WriteFile(cookiesPath, []byte(cookiesContent), 0644); err != nil {
		t.Fatalf("Failed to write test cookies file: %v", err)
	}

	// Create a config.toml with relative cookies_file
	tomlContent := `
[auth]
cookies_file = "auth/my.cookies"
`
	tomlPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config.toml: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify cookies were loaded from the file
	if cfg.Auth.Cookies != "FILE_COOKIES" {
		t.Errorf("Expected cookies 'FILE_COOKIES', got %s", cfg.Auth.Cookies)
	}

	// Verify source tracking includes the resolved cookies path
	if source.CookiesPath == "" {
		t.Error("Expected CookiesPath to be set")
	}
}

// TestLoad_NoConfigFiles verifies Load() works with no configuration files (defaults only)
func TestLoad_NoConfigFiles(t *testing.T) {
	// Create an empty temporary directory
	tmpDir := t.TempDir()

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify source tracking is empty
	if source.TOMLPath != "" {
		t.Error("Expected TOMLPath to be empty")
	}
	if source.CookiesPath != "" {
		t.Error("Expected CookiesPath to be empty")
	}

	// Verify defaults are used
	if cfg.Server.Port != 8115 {
		t.Errorf("Expected default port 8115, got %d", cfg.Server.Port)
	}
	if cfg.P115.ChunkDelay != 2 {
		t.Errorf("Expected default ChunkDelay 2, got %d", cfg.P115.ChunkDelay)
	}
	if cfg.P115.ChunkSize != 200 {
		t.Errorf("Expected default ChunkSize 200, got %d", cfg.P115.ChunkSize)
	}
	if cfg.P115.CooldownMinMs != 1000 {
		t.Errorf("Expected default CooldownMinMs 1000, got %d", cfg.P115.CooldownMinMs)
	}
	if cfg.P115.CooldownMaxMs != 1100 {
		t.Errorf("Expected default CooldownMaxMs 1100, got %d", cfg.P115.CooldownMaxMs)
	}
	if cfg.Proxy.HTTP != "http://127.0.0.1:10809" {
		t.Errorf("Expected default proxy, got %s", cfg.Proxy.HTTP)
	}

	// Verify empty collections
	if len(cfg.RSS) != 0 {
		t.Errorf("Expected empty RSS, got %d entries", len(cfg.RSS))
	}
	// Note: Sites may contain entries from ~/.config/rss2cloud/node-site-config.json or ~/node-site-config.json
	// This is expected behavior as the legacy loader searches in those locations
	// We only verify that RSS is empty since we control that in the temp directory
}

// TestLoad_InvalidTOML verifies Load() returns error for invalid TOML
func TestLoad_InvalidTOML(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create an invalid config.toml
	tomlContent := `
[server]
port = "not-a-number"
`
	tomlPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config.toml: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration should fail
	_, _, err := Load(CLIParams{})
	if err == nil {
		t.Error("Expected Load() to fail with invalid TOML")
	}
}

// TestLoad_InvalidPort verifies Load() returns error for invalid port
func TestLoad_InvalidPort(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a config.toml with invalid port
	tomlContent := `
[server]
port = 99999
`
	tomlPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config.toml: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration should fail
	_, _, err := Load(CLIParams{})
	if err == nil {
		t.Error("Expected Load() to fail with invalid port")
	}
}

// TestLoad_MissingRSSFields verifies Load() returns error for RSS entries missing required fields
func TestLoad_MissingRSSFields(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a config.toml with RSS entry missing required field
	tomlContent := `
[[rss]]
site = "example.com"
url = "http://example.com/rss"
# Missing 'name' field
`
	tomlPath := filepath.Join(tmpDir, "config.toml")
	if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config.toml: %v", err)
	}

	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Load configuration should fail
	_, _, err := Load(CLIParams{})
	if err == nil {
		t.Error("Expected Load() to fail with missing RSS field")
	}
}

func TestLoadWithOptions_SkipsLegacyRSSWhenNotRequested(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "rss.json"), []byte(`{bad json`), 0644); err != nil {
		t.Fatalf("Failed to write malformed rss.json: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	_, _, err := LoadWithOptions(CLIParams{}, LoadOptions{Auth: true})
	if err != nil {
		t.Fatalf("LoadWithOptions() should ignore RSS when not requested, got %v", err)
	}
}

func TestLoadWithOptions_MissingTOMLCookiesFileReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	tomlContent := `
[auth]
cookies_file = "missing.cookies"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatalf("Failed to write config.toml: %v", err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	_, _, err := LoadWithOptions(CLIParams{}, LoadOptions{Auth: true})
	if err == nil {
		t.Fatal("Expected missing cookies_file to return an error")
	}
}

// Helper function to create test RSS config
func createTestRSSConfig(name, url string) RssConfig {
	return RssConfig{
		Name: name,
		Url:  url,
	}
}

// Helper function to create test site config
func createTestSiteConfig(httpsAgent string) SiteConfig {
	return SiteConfig{
		HttpsAgent: httpsAgent,
	}
}
