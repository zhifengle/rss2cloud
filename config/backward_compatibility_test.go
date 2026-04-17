package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBackwardCompatibility_RSSJsonOnly verifies existing rss.json continues working without config.toml
// This test ensures that deployments using only rss.json are not broken by the new config system
func TestBackwardCompatibility_RSSJsonOnly(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	
	// Create only rss.json (no config.toml)
	rssContent := `{
		"mikanani.me": [
			{
				"name": "test anime",
				"url": "https://mikanani.me/RSS/Bangumi?bangumiId=2739",
				"filter": "简体内嵌",
				"cid": "123456",
				"savepath": "动画/测试"
			}
		],
		"share.dmhy.org": [
			{
				"name": "another show",
				"url": "https://share.dmhy.org/topics/rss/rss.xml?keyword=test",
				"filter": "简日双语"
			}
		]
	}`
	rssPath := filepath.Join(tmpDir, "rss.json")
	if err := os.WriteFile(rssPath, []byte(rssContent), 0644); err != nil {
		t.Fatalf("Failed to write test rss.json: %v", err)
	}
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)
	
	// Load configuration - should work without config.toml
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed with legacy rss.json: %v", err)
	}
	
	// Verify no TOML was loaded
	if source.TOMLPath != "" {
		t.Error("Expected TOMLPath to be empty when only rss.json exists")
	}
	
	// Verify RSS configuration was loaded from rss.json
	if len(cfg.RSS) != 2 {
		t.Errorf("Expected 2 RSS sites, got %d", len(cfg.RSS))
	}
	
	// Verify mikanani.me feeds
	mikanFeeds, ok := cfg.RSS["mikanani.me"]
	if !ok {
		t.Fatal("Expected RSS feed for mikanani.me")
	}
	if len(mikanFeeds) != 1 {
		t.Fatalf("Expected 1 feed for mikanani.me, got %d", len(mikanFeeds))
	}
	if mikanFeeds[0].Name != "test anime" {
		t.Errorf("Expected feed name 'test anime', got %s", mikanFeeds[0].Name)
	}
	if mikanFeeds[0].Url != "https://mikanani.me/RSS/Bangumi?bangumiId=2739" {
		t.Errorf("Expected correct URL, got %s", mikanFeeds[0].Url)
	}
	if mikanFeeds[0].Filter != "简体内嵌" {
		t.Errorf("Expected filter '简体内嵌', got %s", mikanFeeds[0].Filter)
	}
	if mikanFeeds[0].Cid != "123456" {
		t.Errorf("Expected cid '123456', got %s", mikanFeeds[0].Cid)
	}
	if mikanFeeds[0].SavePath != "动画/测试" {
		t.Errorf("Expected savepath '动画/测试', got %s", mikanFeeds[0].SavePath)
	}
	
	// Verify dmhy feeds
	dmhyFeeds, ok := cfg.RSS["share.dmhy.org"]
	if !ok {
		t.Fatal("Expected RSS feed for share.dmhy.org")
	}
	if len(dmhyFeeds) != 1 {
		t.Fatalf("Expected 1 feed for share.dmhy.org, got %d", len(dmhyFeeds))
	}
	if dmhyFeeds[0].Name != "another show" {
		t.Errorf("Expected feed name 'another show', got %s", dmhyFeeds[0].Name)
	}
	
	// Verify defaults are used for other settings
	if cfg.Server.Port != 8115 {
		t.Errorf("Expected default port 8115, got %d", cfg.Server.Port)
	}
	if cfg.Proxy.HTTP != "http://127.0.0.1:10809" {
		t.Errorf("Expected default proxy, got %s", cfg.Proxy.HTTP)
	}
	if cfg.P115.ChunkDelay != 2 {
		t.Errorf("Expected default ChunkDelay 2, got %d", cfg.P115.ChunkDelay)
	}
}

// TestBackwardCompatibility_NodeSiteConfigOnly verifies existing node-site-config.json continues working without config.toml
// This test ensures that deployments using only node-site-config.json are not broken by the new config system
func TestBackwardCompatibility_NodeSiteConfigOnly(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	
	// Create only node-site-config.json (no config.toml)
	sitesContent := `{
		"mikanani.me": {
			"httpsAgent": "true"
		},
		"share.dmhy.org": {
			"httpsAgent": "true",
			"headers": {
				"User-Agent": "Mozilla/5.0",
				"Referer": "https://share.dmhy.org"
			}
		},
		"nyaa.si": {
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
	
	// Load configuration - should work without config.toml
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed with legacy node-site-config.json: %v", err)
	}
	
	// Verify no TOML was loaded
	if source.TOMLPath != "" {
		t.Error("Expected TOMLPath to be empty when only node-site-config.json exists")
	}
	
	// Verify sites configuration was loaded from node-site-config.json
	if len(cfg.Sites) != 3 {
		t.Errorf("Expected 3 sites, got %d", len(cfg.Sites))
	}
	
	// Verify mikanani.me config
	mikanConfig, ok := cfg.Sites["mikanani.me"]
	if !ok {
		t.Fatal("Expected site config for mikanani.me")
	}
	if mikanConfig.HttpsAgent != "true" {
		t.Errorf("Expected HttpsAgent 'true', got %s", mikanConfig.HttpsAgent)
	}
	
	// Verify dmhy config with headers
	dmhyConfig, ok := cfg.Sites["share.dmhy.org"]
	if !ok {
		t.Fatal("Expected site config for share.dmhy.org")
	}
	if dmhyConfig.HttpsAgent != "true" {
		t.Errorf("Expected HttpsAgent 'true', got %s", dmhyConfig.HttpsAgent)
	}
	if dmhyConfig.Headers["User-Agent"] != "Mozilla/5.0" {
		t.Errorf("Expected User-Agent 'Mozilla/5.0', got %s", dmhyConfig.Headers["User-Agent"])
	}
	if dmhyConfig.Headers["Referer"] != "https://share.dmhy.org" {
		t.Errorf("Expected Referer 'https://share.dmhy.org', got %s", dmhyConfig.Headers["Referer"])
	}
	
	// Verify nyaa.si config
	nyaaConfig, ok := cfg.Sites["nyaa.si"]
	if !ok {
		t.Fatal("Expected site config for nyaa.si")
	}
	if nyaaConfig.HttpsAgent != "true" {
		t.Errorf("Expected HttpsAgent 'true', got %s", nyaaConfig.HttpsAgent)
	}
	
	// Verify defaults are used for other settings
	if cfg.Server.Port != 8115 {
		t.Errorf("Expected default port 8115, got %d", cfg.Server.Port)
	}
	if cfg.Proxy.HTTP != "http://127.0.0.1:10809" {
		t.Errorf("Expected default proxy, got %s", cfg.Proxy.HTTP)
	}
}

// TestBackwardCompatibility_CookiesOnly verifies existing .cookies continues working without config.toml
// This test ensures that deployments using only .cookies file are not broken by the new config system
func TestBackwardCompatibility_CookiesOnly(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	
	// Create only .cookies file (no config.toml)
	cookiesContent := "UID=123456789; CID=987654321; SEID=abcdef123456; KID=xyz789"
	cookiesPath := filepath.Join(tmpDir, ".cookies")
	if err := os.WriteFile(cookiesPath, []byte(cookiesContent), 0600); err != nil {
		t.Fatalf("Failed to write test .cookies: %v", err)
	}
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)
	
	// Load configuration - should work without config.toml
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed with legacy .cookies: %v", err)
	}
	
	// Verify no TOML was loaded
	if source.TOMLPath != "" {
		t.Error("Expected TOMLPath to be empty when only .cookies exists")
	}
	
	// Verify cookies path was tracked
	if source.CookiesPath == "" {
		t.Error("Expected CookiesPath to be set")
	}
	
	// Verify cookies were loaded from .cookies file
	if cfg.Auth.Cookies != cookiesContent {
		t.Errorf("Expected cookies '%s', got '%s'", cookiesContent, cfg.Auth.Cookies)
	}
	
	// Verify defaults are used for other settings
	if cfg.Server.Port != 8115 {
		t.Errorf("Expected default port 8115, got %d", cfg.Server.Port)
	}
	if cfg.Proxy.HTTP != "http://127.0.0.1:10809" {
		t.Errorf("Expected default proxy, got %s", cfg.Proxy.HTTP)
	}
	if cfg.P115.ChunkDelay != 2 {
		t.Errorf("Expected default ChunkDelay 2, got %d", cfg.P115.ChunkDelay)
	}
	if cfg.P115.ChunkSize != 200 {
		t.Errorf("Expected default ChunkSize 200, got %d", cfg.P115.ChunkSize)
	}
}

// TestBackwardCompatibility_AllLegacyFiles verifies all legacy files work together without config.toml
// This test ensures that existing deployments with all legacy files continue working
func TestBackwardCompatibility_AllLegacyFiles(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	
	// Create rss.json
	rssContent := `{
		"example.com": [
			{
				"name": "legacy-rss-feed",
				"url": "http://example.com/rss"
			}
		]
	}`
	rssPath := filepath.Join(tmpDir, "rss.json")
	if err := os.WriteFile(rssPath, []byte(rssContent), 0644); err != nil {
		t.Fatalf("Failed to write test rss.json: %v", err)
	}
	
	// Create node-site-config.json
	sitesContent := `{
		"example.com": {
			"httpsAgent": "true"
		}
	}`
	sitesPath := filepath.Join(tmpDir, "node-site-config.json")
	if err := os.WriteFile(sitesPath, []byte(sitesContent), 0644); err != nil {
		t.Fatalf("Failed to write test node-site-config.json: %v", err)
	}
	
	// Create .cookies
	cookiesContent := "UID=111; CID=222; SEID=333; KID=444"
	cookiesPath := filepath.Join(tmpDir, ".cookies")
	if err := os.WriteFile(cookiesPath, []byte(cookiesContent), 0600); err != nil {
		t.Fatalf("Failed to write test .cookies: %v", err)
	}
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)
	
	// Load configuration - should work without config.toml
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed with all legacy files: %v", err)
	}
	
	// Verify no TOML was loaded
	if source.TOMLPath != "" {
		t.Error("Expected TOMLPath to be empty when only legacy files exist")
	}
	
	// Verify all legacy configurations were loaded
	if cfg.Auth.Cookies != cookiesContent {
		t.Errorf("Expected cookies from .cookies file, got %s", cfg.Auth.Cookies)
	}
	
	if len(cfg.RSS) != 1 {
		t.Errorf("Expected 1 RSS site from rss.json, got %d", len(cfg.RSS))
	}
	if feeds, ok := cfg.RSS["example.com"]; !ok || len(feeds) != 1 || feeds[0].Name != "legacy-rss-feed" {
		t.Error("Expected RSS feed from rss.json")
	}
	
	if len(cfg.Sites) != 1 {
		t.Errorf("Expected 1 site from node-site-config.json, got %d", len(cfg.Sites))
	}
	if site, ok := cfg.Sites["example.com"]; !ok || site.HttpsAgent != "true" {
		t.Error("Expected site config from node-site-config.json")
	}
	
	// Verify defaults are used for settings not in legacy files
	if cfg.Server.Port != 8115 {
		t.Errorf("Expected default port 8115, got %d", cfg.Server.Port)
	}
	if cfg.Proxy.HTTP != "http://127.0.0.1:10809" {
		t.Errorf("Expected default proxy, got %s", cfg.Proxy.HTTP)
	}
}

// TestBackwardCompatibility_NoConfigFiles verifies program defaults are used when no config files exist
// This test ensures that the program can start with sensible defaults when no configuration is provided
func TestBackwardCompatibility_NoConfigFiles(t *testing.T) {
	// Create an empty temporary directory (no config files at all)
	tmpDir := t.TempDir()
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)
	
	// Load configuration - should work with defaults
	cfg, source, err := Load(CLIParams{})
	if err != nil {
		t.Fatalf("Load() failed with no config files: %v", err)
	}
	
	// Verify no config files were loaded
	if source.TOMLPath != "" {
		t.Error("Expected TOMLPath to be empty when no config files exist")
	}
	if source.CookiesPath != "" {
		t.Error("Expected CookiesPath to be empty when no config files exist")
	}
	
	// Verify all defaults are used
	if cfg.Auth.Cookies != "" {
		t.Errorf("Expected empty cookies, got %s", cfg.Auth.Cookies)
	}
	
	if cfg.Server.Port != 8115 {
		t.Errorf("Expected default port 8115, got %d", cfg.Server.Port)
	}
	
	if cfg.P115.DisableCache != false {
		t.Error("Expected default DisableCache false")
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
		t.Errorf("Expected default proxy 'http://127.0.0.1:10809', got %s", cfg.Proxy.HTTP)
	}
	
	if len(cfg.RSS) != 0 {
		t.Errorf("Expected empty RSS config, got %d entries", len(cfg.RSS))
	}
	
	// Note: Sites may contain entries from ~/.config/rss2cloud/node-site-config.json or ~/node-site-config.json
	// This is expected behavior as the legacy loader searches in those locations
	// We don't verify Sites is empty because it may legitimately have entries from user's home directory
}

// TestBackwardCompatibility_LegacyWithCLIOverride verifies CLI parameters override legacy files
// This test ensures that command-line parameters maintain highest priority even with legacy files
func TestBackwardCompatibility_LegacyWithCLIOverride(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	
	// Create legacy .cookies
	cookiesContent := "LEGACY_COOKIES"
	cookiesPath := filepath.Join(tmpDir, ".cookies")
	if err := os.WriteFile(cookiesPath, []byte(cookiesContent), 0600); err != nil {
		t.Fatalf("Failed to write test .cookies: %v", err)
	}
	
	// Create legacy rss.json
	rssContent := `{
		"legacy.com": [
			{
				"name": "legacy-feed",
				"url": "http://legacy.com/rss"
			}
		]
	}`
	rssPath := filepath.Join(tmpDir, "rss.json")
	if err := os.WriteFile(rssPath, []byte(rssContent), 0644); err != nil {
		t.Fatalf("Failed to write test rss.json: %v", err)
	}
	
	// Create CLI RSS file
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
		Cookies:       "CLI_COOKIES",
		Port:          9999,
		RSSPath:       cliRSSPath,
		ChunkDelay:    10,
		DisableCache:  true,
	})
	if err != nil {
		t.Fatalf("Load() failed with CLI overrides: %v", err)
	}
	
	// Verify CLI parameters override legacy files
	if cfg.Auth.Cookies != "CLI_COOKIES" {
		t.Errorf("Expected CLI cookies to override legacy, got %s", cfg.Auth.Cookies)
	}
	
	if cfg.Server.Port != 9999 {
		t.Errorf("Expected CLI port 9999, got %d", cfg.Server.Port)
	}
	
	if cfg.P115.ChunkDelay != 10 {
		t.Errorf("Expected CLI ChunkDelay 10, got %d", cfg.P115.ChunkDelay)
	}
	
	if !cfg.P115.DisableCache {
		t.Error("Expected CLI DisableCache true")
	}
	
	// Verify CLI RSS path overrides legacy rss.json
	if len(cfg.RSS) != 1 {
		t.Errorf("Expected 1 RSS site from CLI, got %d", len(cfg.RSS))
	}
	if feeds, ok := cfg.RSS["cli.com"]; !ok || len(feeds) != 1 || feeds[0].Name != "cli-feed" {
		t.Error("Expected RSS feed from CLI path, not legacy rss.json")
	}
}
