package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const defaultHistoryFile = ".rss2cloud_shell_history"

// shellHistory holds in-memory command history and manages disk persistence.
type shellHistory struct {
	file    string
	entries []string
}

func newShellHistory(file string) *shellHistory {
	h := &shellHistory{file: file}
	h.load()
	return h
}

// add appends a non-empty, non-duplicate-of-last line to history.
func (h *shellHistory) add(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == line {
		return
	}
	h.entries = append(h.entries, line)
}

// all returns all history entries (oldest first).
func (h *shellHistory) all() []string {
	return h.entries
}

// save writes history to disk, one entry per line.
// Parent directories are created as needed.
func (h *shellHistory) save() error {
	if h.file == "" {
		return nil
	}
	if dir := filepath.Dir(h.file); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(h.file)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, e := range h.entries {
		w.WriteString(e)
		w.WriteByte('\n')
	}
	return w.Flush()
}

// load reads history from disk.
func (h *shellHistory) load() {
	f, err := os.Open(h.file)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			h.entries = append(h.entries, line)
		}
	}
}
