package config

import (
	"testing"
)

// TestResolve_CLIOverridesTOML verifies CLI parameters take priority over TOML
func TestResolve_CLIOverridesTOML(t *testing.T) {
	cli := CLIParams{
		Cookies:       "CLI_COOKIES",
		Port:          9000,
		DisableCache:  true,
		ChunkDelay:    5,
		ChunkSize:     300,
		CooldownMinMs: 2000,
		CooldownMaxMs: 2500,
	}

	toml := &TOMLConfig{
		Auth: TOMLAuthConfig{
			Cookies: "TOML_COOKIES",
		},
		Server: TOMLServerConfig{
			Port: 8080,
		},
		P115: TOMLP115Config{
			DisableCache:  false,
			ChunkDelay:    3,
			ChunkSize:     250,
			CooldownMinMs: 1500,
			CooldownMaxMs: 1800,
		},
	}

	cfg := Resolve(cli, toml, "", nil)

	if cfg.Auth.Cookies != "CLI_COOKIES" {
		t.Errorf("Expected CLI cookies, got %s", cfg.Auth.Cookies)
	}
	if cfg.Server.Port != 9000 {
		t.Errorf("Expected CLI port 9000, got %d", cfg.Server.Port)
	}
	if !cfg.P115.DisableCache {
		t.Error("Expected CLI DisableCache true")
	}
	if cfg.P115.ChunkDelay != 5 {
		t.Errorf("Expected CLI ChunkDelay 5, got %d", cfg.P115.ChunkDelay)
	}
	if cfg.P115.ChunkSize != 300 {
		t.Errorf("Expected CLI ChunkSize 300, got %d", cfg.P115.ChunkSize)
	}
	if cfg.P115.CooldownMinMs != 2000 {
		t.Errorf("Expected CLI CooldownMinMs 2000, got %d", cfg.P115.CooldownMinMs)
	}
	if cfg.P115.CooldownMaxMs != 2500 {
		t.Errorf("Expected CLI CooldownMaxMs 2500, got %d", cfg.P115.CooldownMaxMs)
	}
}

// TestResolve_TOMLOverridesDefaults verifies TOML values override defaults
func TestResolve_TOMLOverridesDefaults(t *testing.T) {
	toml := &TOMLConfig{
		Server: TOMLServerConfig{
			Port: 8080,
		},
		P115: TOMLP115Config{
			ChunkDelay:    3,
			ChunkSize:     250,
			CooldownMinMs: 1500,
			CooldownMaxMs: 1800,
		},
		Proxy: TOMLProxyConfig{
			HTTP: "http://proxy.example.com:8080",
		},
	}

	cfg := Resolve(CLIParams{}, toml, "", nil)

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected TOML port 8080, got %d", cfg.Server.Port)
	}
	if cfg.P115.ChunkDelay != 3 {
		t.Errorf("Expected TOML ChunkDelay 3, got %d", cfg.P115.ChunkDelay)
	}
	if cfg.P115.ChunkSize != 250 {
		t.Errorf("Expected TOML ChunkSize 250, got %d", cfg.P115.ChunkSize)
	}
	if cfg.P115.CooldownMinMs != 1500 {
		t.Errorf("Expected TOML CooldownMinMs 1500, got %d", cfg.P115.CooldownMinMs)
	}
	if cfg.P115.CooldownMaxMs != 1800 {
		t.Errorf("Expected TOML CooldownMaxMs 1800, got %d", cfg.P115.CooldownMaxMs)
	}
	if cfg.Proxy.HTTP != "http://proxy.example.com:8080" {
		t.Errorf("Expected TOML proxy, got %s", cfg.Proxy.HTTP)
	}
}

// TestResolve_Defaults verifies default values are used when no config provided
func TestResolve_Defaults(t *testing.T) {
	cfg := Resolve(CLIParams{}, nil, "", nil)

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
}

// TestResolve_IndependentSectionMerging verifies sections can come from different sources
func TestResolve_IndependentSectionMerging(t *testing.T) {
	cli := CLIParams{
		Cookies: "CLI_COOKIES",
	}

	toml := &TOMLConfig{
		RSS: []TOMLRSSConfig{
			{Site: "example.com", Name: "feed1", URL: "http://example.com/rss"},
		},
	}

	legacy := &LegacyConfig{
		Sites: map[string]SiteConfig{
			"example.com": {HttpsAgent: "true"},
		},
	}

	cfg := Resolve(cli, toml, "", legacy)

	// Auth from CLI
	if cfg.Auth.Cookies != "CLI_COOKIES" {
		t.Errorf("Expected CLI cookies, got %s", cfg.Auth.Cookies)
	}

	// RSS from TOML
	if len(cfg.RSS) != 1 || len(cfg.RSS["example.com"]) != 1 {
		t.Error("Expected RSS from TOML")
	}

	// Sites from legacy
	if len(cfg.Sites) != 1 || cfg.Sites["example.com"].HttpsAgent != "true" {
		t.Error("Expected sites from legacy")
	}
}

// TestResolve_TOMLRSSOverridesLegacy verifies TOML RSS takes priority over legacy
func TestResolve_TOMLRSSOverridesLegacy(t *testing.T) {
	toml := &TOMLConfig{
		RSS: []TOMLRSSConfig{
			{Site: "toml.com", Name: "toml-feed", URL: "http://toml.com/rss"},
		},
	}

	legacy := &LegacyConfig{
		RSS: map[string][]RssConfig{
			"legacy.com": {{Name: "legacy-feed", Url: "http://legacy.com/rss"}},
		},
	}

	cfg := Resolve(CLIParams{}, toml, "", legacy)

	if len(cfg.RSS) != 1 {
		t.Errorf("Expected 1 RSS site, got %d", len(cfg.RSS))
	}
	if _, ok := cfg.RSS["toml.com"]; !ok {
		t.Error("Expected TOML RSS, got legacy")
	}
}

// TestResolve_TOMLSitesOverridesLegacy verifies TOML sites take priority over legacy
func TestResolve_TOMLSitesOverridesLegacy(t *testing.T) {
	toml := &TOMLConfig{
		Sites: map[string]TOMLSiteConfig{
			"toml.com": {HTTPSAgent: true},
		},
	}

	legacy := &LegacyConfig{
		Sites: map[string]SiteConfig{
			"legacy.com": {HttpsAgent: "true"},
		},
	}

	cfg := Resolve(CLIParams{}, toml, "", legacy)

	if len(cfg.Sites) != 1 {
		t.Errorf("Expected 1 site, got %d", len(cfg.Sites))
	}
	if _, ok := cfg.Sites["toml.com"]; !ok {
		t.Error("Expected TOML sites, got legacy")
	}
}
