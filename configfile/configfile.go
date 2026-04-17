package configfile

import (
	"errors"
	"os"
	"path/filepath"
)

const AppDirName = "rss2cloud"

func UserConfigDir() (string, bool) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", false
	}
	return filepath.Join(home, ".config", AppDirName), true
}

func CandidatePaths(filename string, includeLegacyHome bool) []string {
	if filepath.IsAbs(filename) {
		return []string{filename}
	}

	paths := []string{filename}
	if dir, ok := UserConfigDir(); ok {
		paths = append(paths, filepath.Join(dir, filename))
	}
	if includeLegacyHome {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			paths = append(paths, filepath.Join(home, filename))
		}
	}
	return dedupe(paths)
}

func Find(filename string, includeLegacyHome bool) (string, bool) {
	for _, candidate := range CandidatePaths(filename, includeLegacyHome) {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func ReadFile(filename string, includeLegacyHome bool) ([]byte, string, error) {
	path, ok := Find(filename, includeLegacyHome)
	if !ok {
		return nil, "", errors.Join(os.ErrNotExist, errors.New(filename))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, err
	}
	return data, path, nil
}

func ExistingPathOrDefault(filename string) string {
	if path, ok := Find(filename, false); ok {
		return path
	}
	return filename
}

func dedupe(paths []string) []string {
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
