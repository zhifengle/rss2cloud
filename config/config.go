package config

import (
	"encoding/json"
	"os"
)

// RssConfig represents RSS feed configuration (local copy to avoid circular dependency)
type RssConfig struct {
	Name       string `json:"name"`
	Url        string `json:"url"`
	Cid        string `json:"cid,omitempty"`
	SavePath   string `json:"savepath,omitempty"`
	Filter     string `json:"filter,omitempty"`
	Expiration uint   `json:"expiration,omitempty"`
}

// SiteConfig represents site-specific HTTP configuration (local copy to avoid circular dependency)
type SiteConfig struct {
	HttpsAgent string            `json:"httpsAgent,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// P115Option represents 115 cloud storage options (local copy to avoid circular dependency)
type P115Option struct {
	DisableCache  bool
	ChunkDelay    int
	ChunkSize     int
	CooldownMinMs int
	CooldownMaxMs int
}

// Config represents the unified configuration
type Config struct {
	Auth     AuthConfig
	Server   ServerConfig
	Database DatabaseConfig
	P115     P115Config
	Proxy    ProxyConfig
	RSS      map[string][]RssConfig // Keyed by site host
	Sites    map[string]SiteConfig  // Keyed by site host
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Cookies string
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Port int
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Path string
}

// P115Config represents 115 cloud storage configuration
type P115Config struct {
	DisableCache  bool
	ChunkDelay    int
	ChunkSize     int
	CooldownMinMs int
	CooldownMaxMs int
}

// ProxyConfig represents proxy configuration
type ProxyConfig struct {
	HTTP string
}

// ConfigSource tracks the source of loaded configuration for cookie save operations
type ConfigSource struct {
	TOMLPath    string // Path to config.toml if loaded
	CookiesPath string // Path to cookies file that was read
}

// CLIParams captures command-line parameters
type CLIParams struct {
	Cookies          string
	RSSPath          string
	Port             int
	PortSet          bool
	DisableCache     bool
	DisableCacheSet  bool
	ChunkDelay       int
	ChunkDelaySet    bool
	ChunkSize        int
	ChunkSizeSet     bool
	CooldownMinMs    int
	CooldownMinMsSet bool
	CooldownMaxMs    int
	CooldownMaxMsSet bool
}

// LegacyConfig holds loaded legacy configuration
type LegacyConfig struct {
	RSS         map[string][]RssConfig
	Sites       map[string]SiteConfig
	Cookies     string
	CookiesPath string
}

// LoadOptions controls which file-backed sections should be loaded.
type LoadOptions struct {
	RSS   bool
	Sites bool
	Auth  bool
}

func LoadAllOptions() LoadOptions {
	return LoadOptions{RSS: true, Sites: true, Auth: true}
}

// ToP115Option converts P115Config to P115Option
func (c *P115Config) ToP115Option() P115Option {
	return P115Option{
		DisableCache:  c.DisableCache,
		ChunkDelay:    c.ChunkDelay,
		ChunkSize:     c.ChunkSize,
		CooldownMinMs: c.CooldownMinMs,
		CooldownMaxMs: c.CooldownMaxMs,
	}
}

// Resolve merges configuration from multiple sources according to priority rules
// Priority: CLI > TOML > Legacy > Defaults
func Resolve(cli CLIParams, toml *TOMLConfig, tomlPath string, legacy *LegacyConfig) *Config {
	cfg := &Config{
		RSS:   make(map[string][]RssConfig),
		Sites: make(map[string]SiteConfig),
	}

	// Auth priority: CLI cookies > TOML cookies > TOML cookies_file > legacy .cookies
	if cli.Cookies != "" {
		cfg.Auth.Cookies = cli.Cookies
	} else if toml != nil && toml.Auth.Cookies != "" {
		cfg.Auth.Cookies = toml.Auth.Cookies
	} else if toml != nil && toml.Auth.CookiesFile != "" {
		// Read cookies from file specified in TOML
		cookiesPath := ResolveCookiesFile(toml.Auth.CookiesFile, tomlPath)
		if data, err := os.ReadFile(cookiesPath); err == nil {
			cfg.Auth.Cookies = string(data)
		}
	} else if legacy != nil {
		cfg.Auth.Cookies = legacy.Cookies
	}

	// Server priority: CLI port > TOML port > default 8115
	if cli.PortSet || cli.Port != 0 {
		cfg.Server.Port = cli.Port
	} else if toml != nil && toml.Server.Port != 0 {
		cfg.Server.Port = toml.Server.Port
	} else {
		cfg.Server.Port = 8115 // Default
	}

	// Database priority: TOML database.path > existing db.sqlite > default "db.sqlite"
	if toml != nil && toml.Database.Path != "" {
		// Resolve relative path based on config.toml directory
		cfg.Database.Path = ResolveDatabasePath(toml.Database.Path, tomlPath)
	} else if path, ok := findFile("db.sqlite", false); ok {
		cfg.Database.Path = path
	} else {
		cfg.Database.Path = "db.sqlite" // Default
	}

	// P115 priority: CLI params > TOML > defaults
	// DisableCache: CLI or TOML (boolean OR logic)
	cfg.P115.DisableCache = (cli.DisableCacheSet && cli.DisableCache) || (!cli.DisableCacheSet && cli.DisableCache) || (toml != nil && toml.P115.DisableCache)

	// ChunkDelay: CLI > TOML > default 2
	if cli.ChunkDelaySet || cli.ChunkDelay != 0 {
		cfg.P115.ChunkDelay = cli.ChunkDelay
	} else if toml != nil && toml.P115.ChunkDelay != 0 {
		cfg.P115.ChunkDelay = toml.P115.ChunkDelay
	} else {
		cfg.P115.ChunkDelay = 2 // Default
	}

	// ChunkSize: CLI > TOML > default 200
	if cli.ChunkSizeSet || cli.ChunkSize != 0 {
		cfg.P115.ChunkSize = cli.ChunkSize
	} else if toml != nil && toml.P115.ChunkSize != 0 {
		cfg.P115.ChunkSize = toml.P115.ChunkSize
	} else {
		cfg.P115.ChunkSize = 200 // Default
	}

	// CooldownMinMs: CLI > TOML > default 1000
	if cli.CooldownMinMsSet || cli.CooldownMinMs != 0 {
		cfg.P115.CooldownMinMs = cli.CooldownMinMs
	} else if toml != nil && toml.P115.CooldownMinMs != 0 {
		cfg.P115.CooldownMinMs = toml.P115.CooldownMinMs
	} else {
		cfg.P115.CooldownMinMs = 1000 // Default
	}

	// CooldownMaxMs: CLI > TOML > default 1100
	if cli.CooldownMaxMsSet || cli.CooldownMaxMs != 0 {
		cfg.P115.CooldownMaxMs = cli.CooldownMaxMs
	} else if toml != nil && toml.P115.CooldownMaxMs != 0 {
		cfg.P115.CooldownMaxMs = toml.P115.CooldownMaxMs
	} else {
		cfg.P115.CooldownMaxMs = 1100 // Default
	}

	// RSS priority: CLI --rss path > TOML [[rss]] > legacy rss.json
	if cli.RSSPath != "" {
		// Load RSS from CLI-specified path
		// This will be handled by the caller (Load function)
		// For now, we just mark that CLI takes priority
		cfg.RSS = make(map[string][]RssConfig)
	} else if toml != nil && len(toml.RSS) > 0 {
		cfg.RSS = TransformTOMLRSS(toml.RSS)
	} else if legacy != nil && legacy.RSS != nil {
		cfg.RSS = legacy.RSS
	}

	// Sites priority: TOML [sites] > legacy node-site-config.json
	if toml != nil && len(toml.Sites) > 0 {
		cfg.Sites = TransformTOMLSites(toml.Sites)
	} else if legacy != nil && legacy.Sites != nil {
		cfg.Sites = legacy.Sites
	}

	// Proxy priority: TOML [proxy].http > default "http://127.0.0.1:10809"
	if toml != nil && toml.Proxy.HTTP != "" {
		cfg.Proxy.HTTP = toml.Proxy.HTTP
	} else {
		cfg.Proxy.HTTP = "http://127.0.0.1:10809" // Default
	}

	return cfg
}

// Load discovers and loads configuration from all sources
// Returns merged configuration and source tracking information
func Load(cliParams CLIParams) (*Config, *ConfigSource, error) {
	return LoadWithOptions(cliParams, LoadAllOptions())
}

// LoadWithOptions discovers and loads only the requested file-backed sections.
func LoadWithOptions(cliParams CLIParams, options LoadOptions) (*Config, *ConfigSource, error) {
	source := &ConfigSource{}

	// Step 1: Try to find and load config.toml
	var tomlConfig *TOMLConfig
	var tomlPath string
	if path, found := FindConfigFile(); found {
		cfg, err := LoadTOML(path)
		if err != nil {
			return nil, nil, err
		}
		tomlConfig = cfg
		tomlPath = path
		source.TOMLPath = path
	}

	if tomlConfig != nil {
		tomlCopy := *tomlConfig
		if !options.RSS {
			tomlCopy.RSS = nil
		}
		if !options.Sites {
			tomlCopy.Sites = nil
		}
		if !options.Auth {
			tomlCopy.Auth = TOMLAuthConfig{}
		}
		tomlConfig = &tomlCopy
	}

	// Step 2: Load legacy configuration for sections missing in TOML
	legacyConfig := &LegacyConfig{}

	// Load legacy RSS if not in TOML and no CLI override
	if options.RSS && cliParams.RSSPath == "" && (tomlConfig == nil || len(tomlConfig.RSS) == 0) {
		rss, err := LoadLegacyRSS()
		if err != nil {
			return nil, nil, err
		}
		legacyConfig.RSS = rss
	}

	// Load legacy sites if not in TOML
	if options.Sites && (tomlConfig == nil || len(tomlConfig.Sites) == 0) {
		sites, err := LoadLegacySites()
		if err != nil {
			return nil, nil, err
		}
		legacyConfig.Sites = sites
	}

	// Load legacy cookies if not in TOML and no CLI override
	if options.Auth && cliParams.Cookies == "" && (tomlConfig == nil || (tomlConfig.Auth.Cookies == "" && tomlConfig.Auth.CookiesFile == "")) {
		cookies, cookiesPath, err := LoadLegacyCookies()
		if err != nil {
			return nil, nil, err
		}
		legacyConfig.Cookies = cookies
		legacyConfig.CookiesPath = cookiesPath
		source.CookiesPath = cookiesPath
	}

	// Track cookies path from TOML if configured
	if options.Auth && tomlConfig != nil && tomlConfig.Auth.CookiesFile != "" {
		cookiesPath := ResolveCookiesFile(tomlConfig.Auth.CookiesFile, tomlPath)
		data, err := os.ReadFile(cookiesPath)
		if err != nil {
			return nil, nil, err
		}
		tomlConfig.Auth.Cookies = string(data)
		source.CookiesPath = cookiesPath
	}

	// Step 3: Resolve configuration by priority
	cfg := Resolve(cliParams, tomlConfig, tomlPath, legacyConfig)

	// Step 4: Handle CLI RSS path override
	if options.RSS && cliParams.RSSPath != "" {
		// Load RSS from CLI-specified path
		rss, err := LoadRSSFromPath(cliParams.RSSPath)
		if err != nil {
			return nil, nil, err
		}
		cfg.RSS = rss
	}

	return cfg, source, nil
}

// LoadRSSFromPath loads RSS configuration from a specific file path
func LoadRSSFromPath(path string) (map[string][]RssConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config map[string][]RssConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
