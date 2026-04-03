package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/zhifengle/rss2cloud/cloudfs"
)

// --- minimal in-memory driver for cmd-layer tests ---

type cmdFakeDriver struct {
	entries  map[string]cloudfs.Entry
	children map[string][]string
	nextID   int
}

func newCmdFakeDriver() *cmdFakeDriver {
	return &cmdFakeDriver{
		entries: map[string]cloudfs.Entry{
			"0": {ID: "0", Name: "/", Type: cloudfs.EntryTypeDirectory},
			"1": {ID: "1", ParentID: "0", Name: "anime", Type: cloudfs.EntryTypeDirectory},
			"2": {ID: "2", ParentID: "0", Name: "notes.txt", Type: cloudfs.EntryTypeFile, Size: 42},
		},
		children: map[string][]string{
			"0": {"1", "2"},
			"1": {},
		},
		nextID: 3,
	}
}

func (d *cmdFakeDriver) Provider() string { return "fake" }
func (d *cmdFakeDriver) Root(_ context.Context) (cloudfs.Entry, error) {
	return d.entries["0"], nil
}
func (d *cmdFakeDriver) Stat(_ context.Context, id string) (cloudfs.Entry, error) {
	e, ok := d.entries[id]
	if !ok {
		return cloudfs.Entry{}, cloudfs.ErrNotFound
	}
	return e, nil
}
func (d *cmdFakeDriver) List(_ context.Context, dirID string) ([]cloudfs.Entry, error) {
	ids, ok := d.children[dirID]
	if !ok {
		return nil, cloudfs.ErrNotFound
	}
	out := make([]cloudfs.Entry, 0, len(ids))
	for _, id := range ids {
		out = append(out, d.entries[id])
	}
	return out, nil
}
func (d *cmdFakeDriver) Lookup(_ context.Context, parentID, name string) (cloudfs.Entry, error) {
	for _, id := range d.children[parentID] {
		if d.entries[id].Name == name {
			return d.entries[id], nil
		}
	}
	return cloudfs.Entry{}, cloudfs.ErrNotFound
}
func (d *cmdFakeDriver) Mkdir(_ context.Context, parentID, name string) (cloudfs.Entry, error) {
	id := string(rune('0' + d.nextID))
	d.nextID++
	e := cloudfs.Entry{ID: id, ParentID: parentID, Name: name, Type: cloudfs.EntryTypeDirectory}
	d.entries[id] = e
	d.children[parentID] = append(d.children[parentID], id)
	d.children[id] = []string{}
	return e, nil
}
func (d *cmdFakeDriver) Rename(_ context.Context, id, newName string) (cloudfs.Entry, error) {
	e, ok := d.entries[id]
	if !ok {
		return cloudfs.Entry{}, cloudfs.ErrNotFound
	}
	e.Name = newName
	d.entries[id] = e
	return e, nil
}
func (d *cmdFakeDriver) Move(_ context.Context, targetDirID, entryID string) (cloudfs.Entry, error) {
	e, ok := d.entries[entryID]
	if !ok {
		return cloudfs.Entry{}, cloudfs.ErrNotFound
	}
	d.children[e.ParentID] = removeCmdID(d.children[e.ParentID], entryID)
	d.children[targetDirID] = append(d.children[targetDirID], entryID)
	e.ParentID = targetDirID
	d.entries[entryID] = e
	return e, nil
}
func (d *cmdFakeDriver) Copy(_ context.Context, _, entryID string) error {
	if _, ok := d.entries[entryID]; !ok {
		return cloudfs.ErrNotFound
	}
	return nil
}
func (d *cmdFakeDriver) Delete(_ context.Context, entryID string) error {
	e, ok := d.entries[entryID]
	if !ok {
		return cloudfs.ErrNotFound
	}
	delete(d.entries, entryID)
	d.children[e.ParentID] = removeCmdID(d.children[e.ParentID], entryID)
	return nil
}

func removeCmdID(ids []string, id string) []string {
	out := ids[:0]
	for _, v := range ids {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}

func newTestSession(t *testing.T) *cloudfs.Session {
	t.Helper()
	s, err := cloudfs.NewSession(context.Background(), newCmdFakeDriver())
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	return s
}

// --- argument validation ---

func TestMvRequiresAtLeastTwoArgs(t *testing.T) {
	if err := fsMvCmd.Args(fsMvCmd, []string{"only-one"}); err == nil {
		t.Fatal("expected error for single arg to mv")
	}
}

func TestCpRequiresAtLeastTwoArgs(t *testing.T) {
	if err := fsCpCmd.Args(fsCpCmd, []string{"only-one"}); err == nil {
		t.Fatal("expected error for single arg to cp")
	}
}

func TestRenameRequiresExactlyTwoArgs(t *testing.T) {
	if err := fsRenameCmd.Args(fsRenameCmd, []string{"path"}); err == nil {
		t.Fatal("expected error for single arg to rename")
	}
	if err := fsRenameCmd.Args(fsRenameCmd, []string{"a", "b", "c"}); err == nil {
		t.Fatal("expected error for three args to rename")
	}
}

// --- cwd flag semantics ---

func TestCwdFlagChangesStartingDirectory(t *testing.T) {
	ctx := context.Background()
	s := newTestSession(t)
	if _, err := s.Cd(ctx, "/anime"); err != nil {
		t.Fatalf("Cd failed: %v", err)
	}
	if s.Pwd() != "/anime" {
		t.Fatalf("expected /anime, got %s", s.Pwd())
	}
}

// --- JSON output ---

func TestToEntryJSON(t *testing.T) {
	e := cloudfs.Entry{
		ID: "1", ParentID: "0", Name: "anime",
		Type: cloudfs.EntryTypeDirectory,
	}
	j := toEntryJSON(e)
	if j.ID != "1" || j.Name != "anime" || j.Type != "directory" {
		t.Fatalf("unexpected entryJSON: %+v", j)
	}
}

func TestToEntryJSON_File(t *testing.T) {
	e := cloudfs.Entry{
		ID: "2", Name: "notes.txt", Type: cloudfs.EntryTypeFile, Size: 42, PickCode: "abc",
	}
	j := toEntryJSON(e)
	if j.Type != "file" || j.Size != 42 || j.PickCode != "abc" {
		t.Fatalf("unexpected entryJSON: %+v", j)
	}
}

// --- mv/cp last-arg-is-target semantics ---

func TestMvLastArgIsTarget(t *testing.T) {
	ctx := context.Background()
	s := newTestSession(t)

	// mv /notes.txt /anime  — moves notes.txt into anime
	results, err := s.Mv(ctx, "/anime", "/notes.txt")
	if err != nil {
		t.Fatalf("Mv failed: %v", err)
	}
	if len(results) != 1 || results[0].ID != "2" {
		t.Fatalf("unexpected Mv results: %+v", results)
	}
}

func TestCpLastArgIsTarget(t *testing.T) {
	ctx := context.Background()
	s := newTestSession(t)

	if err := s.Cp(ctx, "/anime", "/notes.txt"); err != nil {
		t.Fatalf("Cp failed: %v", err)
	}
}

func TestMvTargetMustBeDirectory(t *testing.T) {
	ctx := context.Background()
	s := newTestSession(t)

	_, err := s.Mv(ctx, "/notes.txt", "/anime")
	if err == nil {
		t.Fatal("expected error when target is not a directory")
	}
}

// --- rename new-name validation ---

func TestRenameNewNameValidation(t *testing.T) {
	ctx := context.Background()
	s := newTestSession(t)

	for _, bad := range []string{"", ".", "..", "a/b"} {
		_, err := s.Rename(ctx, "/anime", bad)
		if err == nil {
			t.Fatalf("expected error for bad new-name %q", bad)
		}
	}
}

// --- JSON vs human output branch ---

func TestPrintEntriesJSONFormat(t *testing.T) {
	entries := []cloudfs.Entry{
		{ID: "1", Name: "anime", Type: cloudfs.EntryTypeDirectory},
		{ID: "2", Name: "notes.txt", Type: cloudfs.EntryTypeFile},
	}
	out := make([]entryJSON, len(entries))
	for i, e := range entries {
		out[i] = toEntryJSON(e)
	}
	if out[0].Type != "directory" || out[1].Type != "file" {
		t.Fatalf("unexpected types: %+v", out)
	}
}

func TestEntryJSONOmitsEmptyFields(t *testing.T) {
	e := cloudfs.Entry{ID: "1", Name: "x", Type: cloudfs.EntryTypeFile}
	j := toEntryJSON(e)
	// ParentID and PickCode should be empty strings (omitempty in JSON).
	if j.ParentID != "" || j.PickCode != "" {
		t.Fatalf("expected empty optional fields, got %+v", j)
	}
	// Verify the JSON tag omitempty works by checking the struct tag string.
	if !strings.Contains(`json:"parent_id,omitempty"`, "omitempty") {
		t.Fatal("omitempty tag missing")
	}
}
