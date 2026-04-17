package config

import (
	"errors"
	"os"
	"path/filepath"
)

const appDirName = "rss2cloud"

func userConfigDir() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	return filepath.Join(home, ".config", appDirName), true
}

func candidatePaths(filename string, includeLegacyHome bool) []string {
	if filepath.IsAbs(filename) {
		return []string{filename}
	}

	paths := []string{filename}
	if dir, ok := userConfigDir(); ok {
		paths = append(paths, filepath.Join(dir, filename))
	}
	if includeLegacyHome {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			paths = append(paths, filepath.Join(home, filename))
		}
	}
	return dedupePaths(paths)
}

func findFile(filename string, includeLegacyHome bool) (string, bool) {
	for _, candidate := range candidatePaths(filename, includeLegacyHome) {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			// Return absolute path for proper resolution
			absPath, err := filepath.Abs(candidate)
			if err != nil {
				return candidate, true
			}
			return absPath, true
		}
	}
	return "", false
}

func readConfigFile(filename string, includeLegacyHome bool) ([]byte, string, error) {
	path, ok := findFile(filename, includeLegacyHome)
	if !ok {
		return nil, "", errors.Join(os.ErrNotExist, errors.New(filename))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}
	return data, path, nil
}

func ExistingCookiePathOrDefault() string {
	if path, ok := findFile(".cookies", false); ok {
		return path
	}
	return ".cookies"
}

func dedupePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		cleaned := filepath.Clean(path)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	return result
}
