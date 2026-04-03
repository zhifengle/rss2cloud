package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/reeflective/readline"
)

const defaultHistoryFile = ".rss2cloud_shell_history"

type readlineHistoryEntry struct {
	DateTime string `json:"datetime"`
	Block    string `json:"block"`
}

func initShellHistory(file string) (readline.History, error) {
	if file == "" {
		return readline.NewInMemoryHistory(), nil
	}

	if dir := filepath.Dir(file); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	legacy, err := isLegacyShellHistoryFile(file)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := os.WriteFile(file, nil, 0o600); err != nil {
			return nil, err
		}
	} else if legacy {
		if err := os.WriteFile(file, nil, 0o600); err != nil {
			return nil, err
		}
	}

	history, err := readline.NewHistoryFromFile(file)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func isLegacyShellHistoryFile(file string) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry readlineHistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return true, nil
		}
		if strings.TrimSpace(entry.Block) == "" || strings.TrimSpace(entry.DateTime) == "" {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}
