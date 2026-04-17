package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLegacyRSS_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create a valid rss.json file
	content := `{
		"mikanani.me": [
			{
				"name": "test feed",
				"url": "https://mikanani.me/RSS/Bangumi?bangumiId=2739",
				"filter": "简体内嵌"
			}
		],
		"nyaa.si": [
			{
				"name": "another feed",
				"url": "https://nyaa.si/rss"
			}
		]
	}`

	err := os.WriteFile("rss.json", []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test rss.json: %v", err)
	}

	result, err := LoadLegacyRSS()
	if err != nil {
		t.Fatalf("LoadLegacyRSS() failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// Check mikanani.me feeds
	mikanFeeds, ok := result["mikanani.me"]
	if !ok {
		t.Fatal("result missing mikanani.me")
	}
	if len(mikanFeeds) != 1 {
		t.Fatalf("len(mikanani.me feeds) = %d, want 1", len(mikanFeeds))
	}
	if mikanFeeds[0].Name != "test feed" {
		t.Errorf("mikanani.me feed name = %q, want %q", mikanFeeds[0].Name, "test feed")
	}

	// Check nyaa.si feeds
	nyaaFeeds, ok := result["nyaa.si"]
	if !ok {
		t.Fatal("result missing nyaa.si")
	}
	if len(nyaaFeeds) != 1 {
		t.Fatalf("len(nyaa.si feeds) = %d, want 1", len(nyaaFeeds))
	}
}

func TestLoadLegacyRSS_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// No rss.json file exists
	result, err := LoadLegacyRSS()
	if err != nil {
		t.Fatalf("LoadLegacyRSS() should not error when file not found: %v", err)
	}

	if result == nil {
		t.Fatal("LoadLegacyRSS() returned nil, want empty map")
	}

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestLoadLegacyRSS_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create invalid JSON
	content := `{invalid json`
	err := os.WriteFile("rss.json", []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test rss.json: %v", err)
	}

	_, err = LoadLegacyRSS()
	if err == nil {
		t.Error("LoadLegacyRSS() should fail with invalid JSON")
	}
}

func TestLoadLegacySites_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create a valid node-site-config.json file
	// Note: JSON field name is "httpsAgent" (camelCase), not "https_agent"
	content := `{
		"mikanani.me": {
			"httpsAgent": "true"
		},
		"nyaa.si": {
			"httpsAgent": "true",
			"headers": {
				"User-Agent": "Custom UA"
			}
		}
	}`

	err := os.WriteFile("node-site-config.json", []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test node-site-config.json: %v", err)
	}

	result, err := LoadLegacySites()
	if err != nil {
		t.Fatalf("LoadLegacySites() failed: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}

	// Check mikanani.me config
	mikanConfig, ok := result["mikanani.me"]
	if !ok {
		t.Fatal("result missing mikanani.me")
	}
	if mikanConfig.HttpsAgent != "true" {
		t.Errorf("mikanani.me HttpsAgent = %q, want %q", mikanConfig.HttpsAgent, "true")
	}

	// Check nyaa.si config
	nyaaConfig, ok := result["nyaa.si"]
	if !ok {
		t.Fatal("result missing nyaa.si")
	}
	if nyaaConfig.HttpsAgent != "true" {
		t.Errorf("nyaa.si HttpsAgent = %q, want %q", nyaaConfig.HttpsAgent, "true")
	}
	if nyaaConfig.Headers["User-Agent"] != "Custom UA" {
		t.Errorf("nyaa.si User-Agent = %q, want %q", nyaaConfig.Headers["User-Agent"], "Custom UA")
	}
}

func TestLoadLegacySites_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Set HOME to tmpDir to isolate from real config files
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	
	// Create an isolated work directory
	workDir := filepath.Join(tmpDir, "work")
	err := os.Mkdir(workDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create work directory: %v", err)
	}
	os.Chdir(workDir)

	// No node-site-config.json file exists
	result, err := LoadLegacySites()
	if err != nil {
		t.Fatalf("LoadLegacySites() should not error when file not found: %v", err)
	}

	if result == nil {
		t.Fatal("LoadLegacySites() returned nil, want empty map")
	}

	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestLoadLegacySites_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create invalid JSON
	content := `{invalid json`
	err := os.WriteFile("node-site-config.json", []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test node-site-config.json: %v", err)
	}

	_, err = LoadLegacySites()
	if err == nil {
		t.Error("LoadLegacySites() should fail with invalid JSON")
	}
}

func TestLoadLegacyCookies_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create a .cookies file
	cookieContent := "UID=123; CID=456; SEID=789; KID=abc"
	err := os.WriteFile(".cookies", []byte(cookieContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test .cookies: %v", err)
	}

	cookies, path, err := LoadLegacyCookies()
	if err != nil {
		t.Fatalf("LoadLegacyCookies() failed: %v", err)
	}

	if cookies != cookieContent {
		t.Errorf("cookies = %q, want %q", cookies, cookieContent)
	}

	// Path should be the absolute path to .cookies in current directory
	expectedPath, _ := filepath.Abs(".cookies")
	if path != expectedPath {
		t.Errorf("path = %q, want %q", path, expectedPath)
	}
}

func TestLoadLegacyCookies_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// No .cookies file exists
	cookies, path, err := LoadLegacyCookies()
	if err != nil {
		t.Fatalf("LoadLegacyCookies() should not error when file not found: %v", err)
	}

	if cookies != "" {
		t.Errorf("cookies = %q, want empty string", cookies)
	}

	if path != "" {
		t.Errorf("path = %q, want empty string", path)
	}
}

func TestLoadLegacyCookies_PermissionDenied(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("OS") == "Windows_NT" {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create a .cookies file with no read permissions
	err := os.WriteFile(".cookies", []byte("test"), 0000)
	if err != nil {
		t.Fatalf("Failed to create test .cookies: %v", err)
	}
	defer os.Chmod(".cookies", 0644) // Restore permissions for cleanup

	_, _, err = LoadLegacyCookies()
	if err == nil {
		t.Error("LoadLegacyCookies() should fail with permission denied")
	}
}
