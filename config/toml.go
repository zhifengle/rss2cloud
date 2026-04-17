package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// TOMLConfig represents the structure of config.toml
type TOMLConfig struct {
	Auth     TOMLAuthConfig            `toml:"auth"`
	Server   TOMLServerConfig          `toml:"server"`
	Database TOMLDatabaseConfig        `toml:"database"`
	P115     TOMLP115Config            `toml:"p115"`
	Proxy    TOMLProxyConfig           `toml:"proxy"`
	RSS      []TOMLRSSConfig           `toml:"rss"`
	Sites    map[string]TOMLSiteConfig `toml:"sites"`
}

// TOMLAuthConfig represents authentication configuration
type TOMLAuthConfig struct {
	CookiesFile string `toml:"cookies_file"` // Relative to config.toml directory
	Cookies     string `toml:"cookies"`      // Direct cookie string
}

// TOMLServerConfig represents server configuration
type TOMLServerConfig struct {
	Port int `toml:"port"` // Valid range: 1-65535
}

// TOMLDatabaseConfig represents database configuration
type TOMLDatabaseConfig struct {
	Path string `toml:"path"` // SQLite database file path
}

// TOMLP115Config represents 115 cloud storage configuration
type TOMLP115Config struct {
	DisableCache  bool `toml:"disable_cache"`
	ChunkDelay    int  `toml:"chunk_delay"`
	ChunkSize     int  `toml:"chunk_size"`
	CooldownMinMs int  `toml:"cooldown_min_ms"`
	CooldownMaxMs int  `toml:"cooldown_max_ms"`
}

// TOMLProxyConfig represents proxy configuration
type TOMLProxyConfig struct {
	HTTP string `toml:"http"` // HTTP proxy URL
}

// TOMLRSSConfig represents a single RSS feed configuration
type TOMLRSSConfig struct {
	Site       string `toml:"site"`       // Required: host for grouping
	Name       string `toml:"name"`       // Required: feed name
	URL        string `toml:"url"`        // Required: RSS feed URL
	Cid        string `toml:"cid"`        // Optional: 115 directory ID
	SavePath   string `toml:"savepath"`   // Optional: save path
	Filter     string `toml:"filter"`     // Optional: content filter
	Expiration uint   `toml:"expiration"` // Optional: cache expiration
}

// TOMLSiteConfig represents site-specific HTTP configuration
type TOMLSiteConfig struct {
	HTTPSAgent bool              `toml:"https_agent"` // Enable proxy for this site
	Headers    map[string]string `toml:"headers"`     // Custom HTTP headers
}

// FindConfigFile discovers config.toml in standard locations
// Returns path and true if found, empty string and false otherwise
func FindConfigFile() (string, bool) {
	return findFile("config.toml", false)
}

// LoadTOML parses config.toml from the given path
func LoadTOML(path string) (*TOMLConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config.toml at %s: %w", path, err)
	}

	var cfg TOMLConfig
	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config.toml at %s: %w", path, err)
	}

	// Validate the parsed configuration
	if err := validateTOMLConfig(&cfg, path); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateTOMLConfig validates the parsed TOML configuration
func validateTOMLConfig(cfg *TOMLConfig, path string) error {
	// Validate server port range
	if cfg.Server.Port != 0 && (cfg.Server.Port < 1 || cfg.Server.Port > 65535) {
		return fmt.Errorf("invalid server port %d in %s: must be between 1 and 65535", cfg.Server.Port, path)
	}

	// Validate proxy URL format
	if cfg.Proxy.HTTP != "" {
		u, err := url.Parse(cfg.Proxy.HTTP)
		if err != nil {
			return fmt.Errorf("invalid proxy URL %q in %s: %w", cfg.Proxy.HTTP, path, err)
		}
		// Ensure it has a scheme (http:// or https://)
		if u.Scheme == "" {
			return fmt.Errorf("invalid proxy URL %q in %s: missing scheme (http:// or https://)", cfg.Proxy.HTTP, path)
		}
	}

	// Validate RSS required fields
	for i, rss := range cfg.RSS {
		if rss.Site == "" {
			return fmt.Errorf("RSS entry %d in %s missing required field 'site'", i, path)
		}
		if rss.Name == "" {
			return fmt.Errorf("RSS entry %d in %s missing required field 'name'", i, path)
		}
		if rss.URL == "" {
			return fmt.Errorf("RSS entry %d in %s missing required field 'url'", i, path)
		}
		// Validate RSS URL format
		u, err := url.Parse(rss.URL)
		if err != nil {
			return fmt.Errorf("invalid RSS URL %q in entry %d of %s: %w", rss.URL, i, path, err)
		}
		// Ensure it has a scheme (http:// or https://)
		if u.Scheme == "" {
			return fmt.Errorf("invalid RSS URL %q in entry %d of %s: missing scheme (http:// or https://)", rss.URL, i, path)
		}
	}

	return nil
}

// TransformTOMLRSS converts []TOMLRSSConfig to map[string][]RssConfig grouped by site
func TransformTOMLRSS(tomlRSS []TOMLRSSConfig) map[string][]RssConfig {
	result := make(map[string][]RssConfig)

	for _, tr := range tomlRSS {
		rssConfig := RssConfig{
			Name:       tr.Name,
			Url:        tr.URL,
			Cid:        tr.Cid,
			SavePath:   tr.SavePath,
			Filter:     tr.Filter,
			Expiration: tr.Expiration,
		}

		result[tr.Site] = append(result[tr.Site], rssConfig)
	}

	return result
}

// TransformTOMLSites converts map[string]TOMLSiteConfig to map[string]SiteConfig
func TransformTOMLSites(tomlSites map[string]TOMLSiteConfig) map[string]SiteConfig {
	result := make(map[string]SiteConfig)

	for host, ts := range tomlSites {
		siteConfig := SiteConfig{
			Headers: ts.Headers,
		}

		// Convert boolean HTTPSAgent to string for backward compatibility
		if ts.HTTPSAgent {
			siteConfig.HttpsAgent = "true"
		}

		result[host] = siteConfig
	}

	return result
}

// ResolveCookiesFile resolves cookies_file path relative to config.toml directory
// If the path is absolute, it is returned as-is
// If the path is relative, it is resolved relative to the directory containing config.toml
func ResolveCookiesFile(cookiesFile string, tomlPath string) string {
	if cookiesFile == "" {
		return ""
	}

	if filepath.IsAbs(cookiesFile) || strings.HasPrefix(cookiesFile, "/") {
		return cookiesFile
	}

	if tomlPath != "" {
		if strings.Contains(tomlPath, "/") && !strings.Contains(tomlPath, "\\") {
			return path.Join(path.Dir(tomlPath), cookiesFile)
		}
		return filepath.Join(filepath.Dir(tomlPath), cookiesFile)
	}

	return cookiesFile
}

// ResolveDatabasePath resolves database path relative to config.toml directory
// If the path is absolute, it is returned as-is
// If the path is relative, it is resolved relative to the directory containing config.toml
func ResolveDatabasePath(dbPath string, tomlPath string) string {
	if dbPath == "" {
		return ""
	}

	if filepath.IsAbs(dbPath) {
		return dbPath
	}

	if tomlPath != "" {
		return filepath.Join(filepath.Dir(tomlPath), dbPath)
	}

	return dbPath
}
