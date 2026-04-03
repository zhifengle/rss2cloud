package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// shellStateFile is the default path for persisted shell state.
const shellStateFile = ".rss2cloud_shell_state.json"

// shellPersistedState holds the fields written to disk between sessions.
type shellPersistedState struct {
	LastCwd string `json:"last_cwd,omitempty"`
}

// loadShellState reads persisted state from file.
// Returns zero value if the file does not exist or cannot be parsed.
func loadShellState(file string) shellPersistedState {
	data, err := os.ReadFile(file)
	if err != nil {
		return shellPersistedState{}
	}
	var s shellPersistedState
	if err := json.Unmarshal(data, &s); err != nil {
		return shellPersistedState{}
	}
	return s
}

// saveShellState writes state to file, creating parent directories as needed.
func saveShellState(file string, s shellPersistedState) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(file, data, 0o644)
}
