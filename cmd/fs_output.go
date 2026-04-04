package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/zhifengle/rss2cloud/cloudfs"
)

// entryJSON is the JSON-serialisable form of a cloudfs.Entry.
type entryJSON struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id,omitempty"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
	PickCode string `json:"pick_code,omitempty"`
}

func toEntryJSON(e cloudfs.Entry) entryJSON {
	return entryJSON{
		ID:       e.ID,
		ParentID: e.ParentID,
		Name:     e.Name,
		Type:     string(e.Type),
		Size:     e.Size,
		PickCode: e.PickCode,
	}
}

// printEntry prints a single entry in the active output format.
// For shell output, use fprintEntry(out, e) directly.
func printEntry(e cloudfs.Entry) {
	if fsJSON {
		printJSON(toEntryJSON(e))
		return
	}
	fprintEntry(os.Stdout, e)
}

// printEntries prints a slice of entries.
// For shell output, use fprintEntries(out, entries) directly.
func printEntries(entries []cloudfs.Entry) {
	if fsJSON {
		out := make([]entryJSON, len(entries))
		for i, e := range entries {
			out[i] = toEntryJSON(e)
		}
		printJSON(out)
		return
	}
	fprintEntries(os.Stdout, entries)
}

// printJSON marshals v to stdout as indented JSON.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v) //nolint:errcheck
}

// printFsError writes a user-friendly error message to stderr.
func printFsError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
}

// fprintEntry writes a single entry to out for shared one-shot and shell output.
func fprintEntry(out io.Writer, e cloudfs.Entry) {
	fmt.Fprintf(out, "id:        %s\n", e.ID)
	if e.ParentID != "" {
		fmt.Fprintf(out, "parent_id: %s\n", e.ParentID)
	}
	fmt.Fprintf(out, "name:      %s\n", e.Name)
	fmt.Fprintf(out, "type:      %s\n", e.Type)
	fmt.Fprintf(out, "size:      %d\n", e.Size)
	if e.PickCode != "" {
		fmt.Fprintf(out, "pick_code: %s\n", e.PickCode)
	}
}

// fprintEntries writes a list of entries to out for shared one-shot and shell output.
func fprintEntries(out io.Writer, entries []cloudfs.Entry) {
	for _, e := range entries {
		typeChar := "-"
		if e.IsDir() {
			typeChar = "d"
		}
		fmt.Fprintf(out, "%s  %-12s  %s\n", typeChar, e.ID, e.Name)
	}
}
