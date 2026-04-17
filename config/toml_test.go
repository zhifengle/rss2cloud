package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindConfigFile_NotFound(t *testing.T) {
	// Change to a temporary directory where config.toml doesn't exist
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	path, found := FindConfigFile()
	if found {
		t.Errorf("FindConfigFile() found file when none should exist: %s", path)
	}
	if path != "" {
		t.Errorf("FindConfigFile() returned non-empty path when not found: %s", path)
	}
}

func TestFindConfigFile_CurrentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create config.toml in current directory
	configPath := filepath.Join(tmpDir, "config.toml")
	err := os.WriteFile(configPath, []byte("[server]\nport = 8115\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config.toml: %v", err)
	}

	path, found := FindConfigFile()
	if !found {
		t.Error("FindConfigFile() did not find config.toml in current directory")
	}
	if path == "" {
		t.Error("FindConfigFile() returned empty path when file exists")
	}
}

func TestLoadTOML_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `
[auth]
cookies = "UID=123; CID=456"

[server]
port = 8080

[p115]
disable_cache = true
chunk_delay = 3
chunk_size = 300
cooldown_min_ms = 2000
cooldown_max_ms = 2100

[proxy]
http = "http://127.0.0.1:7890"

[[rss]]
site = "mikanani.me"
name = "test feed"
url = "https://mikanani.me/RSS/Bangumi?bangumiId=2739"
filter = "简体内嵌"

[sites."mikanani.me"]
https_agent = true
`

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config.toml: %v", err)
	}

	cfg, err := LoadTOML(configPath)
	if err != nil {
		t.Fatalf("LoadTOML() failed: %v", err)
	}

	// Verify auth section
	if cfg.Auth.Cookies != "UID=123; CID=456" {
		t.Errorf("Auth.Cookies = %q, want %q", cfg.Auth.Cookies, "UID=123; CID=456")
	}

	// Verify server section
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8080)
	}

	// Verify p115 section
	if !cfg.P115.DisableCache {
		t.Error("P115.DisableCache = false, want true")
	}
	if cfg.P115.ChunkDelay != 3 {
		t.Errorf("P115.ChunkDelay = %d, want %d", cfg.P115.ChunkDelay, 3)
	}

	// Verify proxy section
	if cfg.Proxy.HTTP != "http://127.0.0.1:7890" {
		t.Errorf("Proxy.HTTP = %q, want %q", cfg.Proxy.HTTP, "http://127.0.0.1:7890")
	}

	// Verify RSS section
	if len(cfg.RSS) != 1 {
		t.Fatalf("len(RSS) = %d, want %d", len(cfg.RSS), 1)
	}
	if cfg.RSS[0].Site != "mikanani.me" {
		t.Errorf("RSS[0].Site = %q, want %q", cfg.RSS[0].Site, "mikanani.me")
	}
	if cfg.RSS[0].Name != "test feed" {
		t.Errorf("RSS[0].Name = %q, want %q", cfg.RSS[0].Name, "test feed")
	}

	// Verify sites section
	if len(cfg.Sites) != 1 {
		t.Fatalf("len(Sites) = %d, want %d", len(cfg.Sites), 1)
	}
	if !cfg.Sites["mikanani.me"].HTTPSAgent {
		t.Error("Sites[mikanani.me].HTTPSAgent = false, want true")
	}
}

func TestLoadTOML_InvalidSyntax(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Invalid TOML syntax
	content := `
[server
port = 8080
`

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config.toml: %v", err)
	}

	_, err = LoadTOML(configPath)
	if err == nil {
		t.Error("LoadTOML() should fail with invalid syntax")
	}
}

func TestLoadTOML_InvalidPort(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `
[server]
port = 99999
`

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config.toml: %v", err)
	}

	_, err = LoadTOML(configPath)
	if err == nil {
		t.Error("LoadTOML() should fail with invalid port")
	}
}

func TestLoadTOML_InvalidProxyURL(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `
[proxy]
http = "not a valid url with spaces"
`

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config.toml: %v", err)
	}

	_, err = LoadTOML(configPath)
	if err == nil {
		t.Error("LoadTOML() should fail with invalid proxy URL")
	}
}

func TestLoadTOML_MissingRSSFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing site",
			content: `
[[rss]]
name = "test"
url = "https://example.com/rss"
`,
		},
		{
			name: "missing name",
			content: `
[[rss]]
site = "example.com"
url = "https://example.com/rss"
`,
		},
		{
			name: "missing url",
			content: `
[[rss]]
site = "example.com"
name = "test"
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.toml")

			err := os.WriteFile(configPath, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config.toml: %v", err)
			}

			_, err = LoadTOML(configPath)
			if err == nil {
				t.Errorf("LoadTOML() should fail with %s", tt.name)
			}
		})
	}
}

func TestLoadTOML_InvalidRSSURL(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	content := `
[[rss]]
site = "example.com"
name = "test"
url = "not a valid url"
`

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config.toml: %v", err)
	}

	_, err = LoadTOML(configPath)
	if err == nil {
		t.Error("LoadTOML() should fail with invalid RSS URL")
	}
}

func TestLoadTOML_MissingOptionalSections(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Minimal valid config with only server section
	content := `
[server]
port = 8115
`

	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config.toml: %v", err)
	}

	cfg, err := LoadTOML(configPath)
	if err != nil {
		t.Fatalf("LoadTOML() should succeed with missing optional sections: %v", err)
	}

	if cfg.Server.Port != 8115 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 8115)
	}
}

func TestTransformTOMLRSS_GroupsBySite(t *testing.T) {
	tomlRSS := []TOMLRSSConfig{
		{
			Site:   "mikanani.me",
			Name:   "feed1",
			URL:    "https://mikanani.me/RSS/1",
			Filter: "简体",
		},
		{
			Site:       "mikanani.me",
			Name:       "feed2",
			URL:        "https://mikanani.me/RSS/2",
			Cid:        "123",
			SavePath:   "/path",
			Expiration: 3600,
		},
		{
			Site: "nyaa.si",
			Name: "feed3",
			URL:  "https://nyaa.si/rss",
		},
	}

	result := TransformTOMLRSS(tomlRSS)

	// Check that we have 2 sites
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// Check mikanani.me has 2 feeds
	mikanFeeds, ok := result["mikanani.me"]
	if !ok {
		t.Fatal("result missing mikanani.me")
	}
	if len(mikanFeeds) != 2 {
		t.Fatalf("len(mikanani.me feeds) = %d, want 2", len(mikanFeeds))
	}

	// Check nyaa.si has 1 feed
	nyaaFeeds, ok := result["nyaa.si"]
	if !ok {
		t.Fatal("result missing nyaa.si")
	}
	if len(nyaaFeeds) != 1 {
		t.Fatalf("len(nyaa.si feeds) = %d, want 1", len(nyaaFeeds))
	}
}

func TestTransformTOMLRSS_PreservesAllFields(t *testing.T) {
	tomlRSS := []TOMLRSSConfig{
		{
			Site:       "example.com",
			Name:       "test feed",
			URL:        "https://example.com/rss",
			Cid:        "456",
			SavePath:   "/save/path",
			Filter:     "filter text",
			Expiration: 7200,
		},
	}

	result := TransformTOMLRSS(tomlRSS)

	feeds := result["example.com"]
	if len(feeds) != 1 {
		t.Fatalf("len(feeds) = %d, want 1", len(feeds))
	}

	feed := feeds[0]
	if feed.Name != "test feed" {
		t.Errorf("Name = %q, want %q", feed.Name, "test feed")
	}
	if feed.Url != "https://example.com/rss" {
		t.Errorf("Url = %q, want %q", feed.Url, "https://example.com/rss")
	}
	if feed.Cid != "456" {
		t.Errorf("Cid = %q, want %q", feed.Cid, "456")
	}
	if feed.SavePath != "/save/path" {
		t.Errorf("SavePath = %q, want %q", feed.SavePath, "/save/path")
	}
	if feed.Filter != "filter text" {
		t.Errorf("Filter = %q, want %q", feed.Filter, "filter text")
	}
	if feed.Expiration != 7200 {
		t.Errorf("Expiration = %d, want %d", feed.Expiration, 7200)
	}
}

func TestTransformTOMLSites_ConvertsToSiteConfig(t *testing.T) {
	tomlSites := map[string]TOMLSiteConfig{
		"example.com": {
			HTTPSAgent: true,
		},
		"test.com": {
			HTTPSAgent: false,
		},
	}

	result := TransformTOMLSites(tomlSites)

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// Check example.com has https_agent enabled
	exampleConfig, ok := result["example.com"]
	if !ok {
		t.Fatal("result missing example.com")
	}
	if exampleConfig.HttpsAgent != "true" {
		t.Errorf("example.com HttpsAgent = %q, want %q", exampleConfig.HttpsAgent, "true")
	}

	// Check test.com has https_agent disabled (empty string)
	testConfig, ok := result["test.com"]
	if !ok {
		t.Fatal("result missing test.com")
	}
	if testConfig.HttpsAgent != "" {
		t.Errorf("test.com HttpsAgent = %q, want empty string", testConfig.HttpsAgent)
	}
}

func TestTransformTOMLSites_HandlesHeaders(t *testing.T) {
	tomlSites := map[string]TOMLSiteConfig{
		"example.com": {
			HTTPSAgent: true,
			Headers: map[string]string{
				"Cookie":     "session=abc123",
				"User-Agent": "Custom UA",
			},
		},
	}

	result := TransformTOMLSites(tomlSites)

	exampleConfig := result["example.com"]
	if exampleConfig.Headers == nil {
		t.Fatal("Headers is nil")
	}
	if len(exampleConfig.Headers) != 2 {
		t.Fatalf("len(Headers) = %d, want 2", len(exampleConfig.Headers))
	}
	if exampleConfig.Headers["Cookie"] != "session=abc123" {
		t.Errorf("Headers[Cookie] = %q, want %q", exampleConfig.Headers["Cookie"], "session=abc123")
	}
	if exampleConfig.Headers["User-Agent"] != "Custom UA" {
		t.Errorf("Headers[User-Agent] = %q, want %q", exampleConfig.Headers["User-Agent"], "Custom UA")
	}
}

func TestResolveCookiesFile_RelativePath(t *testing.T) {
	tests := []struct {
		name        string
		cookiesFile string
		tomlPath    string
		want        string
	}{
		{
			name:        "relative path with unix separator",
			cookiesFile: ".cookies",
			tomlPath:    "/home/user/.config/rss2cloud/config.toml",
			want:        "/home/user/.config/rss2cloud/.cookies",
		},
		{
			name:        "relative path with subdirectory",
			cookiesFile: "auth/.cookies",
			tomlPath:    "/home/user/.config/rss2cloud/config.toml",
			want:        "/home/user/.config/rss2cloud/auth/.cookies",
		},
		{
			name:        "relative path in current directory",
			cookiesFile: ".cookies",
			tomlPath:    "config.toml",
			want:        ".cookies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCookiesFile(tt.cookiesFile, tt.tomlPath)
			if got != tt.want {
				t.Errorf("ResolveCookiesFile(%q, %q) = %q, want %q", tt.cookiesFile, tt.tomlPath, got, tt.want)
			}
		})
	}
}

func TestResolveCookiesFile_AbsolutePath(t *testing.T) {
	tests := []struct {
		name        string
		cookiesFile string
		tomlPath    string
		want        string
	}{
		{
			name:        "absolute unix path",
			cookiesFile: "/etc/rss2cloud/.cookies",
			tomlPath:    "/home/user/.config/rss2cloud/config.toml",
			want:        "/etc/rss2cloud/.cookies",
		},
		{
			name:        "absolute windows path",
			cookiesFile: "C:/Users/user/.cookies",
			tomlPath:    "C:/Users/user/.config/rss2cloud/config.toml",
			want:        "C:/Users/user/.cookies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveCookiesFile(tt.cookiesFile, tt.tomlPath)
			if got != tt.want {
				t.Errorf("ResolveCookiesFile(%q, %q) = %q, want %q", tt.cookiesFile, tt.tomlPath, got, tt.want)
			}
		})
	}
}

func TestResolveCookiesFile_EmptyPath(t *testing.T) {
	got := ResolveCookiesFile("", "/home/user/config.toml")
	if got != "" {
		t.Errorf("ResolveCookiesFile with empty cookiesFile should return empty string, got %q", got)
	}
}
