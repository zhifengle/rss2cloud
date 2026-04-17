package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// LoadLegacyRSS reads rss.json using existing search rules
// Returns empty config (not error) when file doesn't exist
func LoadLegacyRSS() (map[string][]RssConfig, error) {
	data, _, err := readConfigFile("rss.json", false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string][]RssConfig), nil // Empty config, not an error
		}
		return nil, fmt.Errorf("failed to read rss.json: %w", err)
	}

	var config map[string][]RssConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rss.json: %w", err)
	}

	return config, nil
}

// LoadLegacySites reads node-site-config.json using existing search rules
// Returns empty config (not error) when file doesn't exist
func LoadLegacySites() (map[string]SiteConfig, error) {
	data, _, err := readConfigFile("node-site-config.json", true)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return make(map[string]SiteConfig), nil // Empty config, not an error
		}
		return nil, fmt.Errorf("failed to read node-site-config.json: %w", err)
	}

	var config map[string]SiteConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse node-site-config.json: %w", err)
	}

	return config, nil
}

// LoadLegacyCookies reads .cookies file using existing search rules
// Returns cookies string and path, or empty string and empty path if not found
func LoadLegacyCookies() (string, string, error) {
	data, path, err := readConfigFile(".cookies", false)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", nil // Empty cookies, not an error
		}
		return "", "", fmt.Errorf("failed to read .cookies: %w", err)
	}

	return string(data), path, nil
}
