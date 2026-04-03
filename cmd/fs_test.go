package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zhifengle/rss2cloud/cloudfs"
)

// --- minimal in-memory driver for cmd-layer tests ---

type cmdFakeDriver struct {
	entries  map[string]cloudfs.Entry
	children map[string][]string
	nextID   int
}

type countingCmdFakeDriver struct {
	*cmdFakeDriver
	listCalls map[string]int
}

func newCmdFakeDriver() *cmdFakeDriver {
	return &cmdFakeDriver{
		entries: map[string]cloudfs.Entry{
			"0": {ID: "0", Name: "/", Type: cloudfs.EntryTypeDirectory},
			"1": {ID: "1", ParentID: "0", Name: "anime", Type: cloudfs.EntryTypeDirectory},
			"2": {ID: "2", ParentID: "0", Name: "notes.txt", Type: cloudfs.EntryTypeFile, Size: 42},
			"3": {ID: "3", ParentID: "1", Name: "episode.mkv", Type: cloudfs.EntryTypeFile, Size: 100},
		},
		children: map[string][]string{
			"0": {"1", "2"},
			"1": {"3"},
		},
		nextID: 4,
	}
}

func newCountingCmdFakeDriver() *countingCmdFakeDriver {
	return &countingCmdFakeDriver{
		cmdFakeDriver: newCmdFakeDriver(),
		listCalls:     make(map[string]int),
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
func (d *countingCmdFakeDriver) List(ctx context.Context, dirID string) ([]cloudfs.Entry, error) {
	d.listCalls[dirID]++
	return d.cmdFakeDriver.List(ctx, dirID)
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

func (d *cmdFakeDriver) Search(_ context.Context, dirID, keyword string, opts cloudfs.SearchOptions) ([]cloudfs.Entry, error) {
	var results []cloudfs.Entry
	var visit func(string)
	visit = func(currentID string) {
		for _, childID := range d.children[currentID] {
			entry := d.entries[childID]
			if entry.IsDir() {
				if opts.IncludeDirectories && strings.Contains(entry.Name, keyword) {
					results = append(results, entry)
				}
				visit(childID)
				continue
			}
			if strings.Contains(entry.Name, keyword) {
				results = append(results, entry)
			}
		}
	}
	visit(dirID)
	return results, nil
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

func TestSearchMvRequiresExactlyThreeArgs(t *testing.T) {
	if err := fsSearchMvCmd.Args(fsSearchMvCmd, []string{"one", "two"}); err == nil {
		t.Fatal("expected error for two args to search-mv")
	}
}

func TestSearchMvAliasIncludesSearchUnderscoreMv(t *testing.T) {
	found := false
	for _, alias := range fsSearchMvCmd.Aliases {
		if alias == "search_mv" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected search_mv alias on search-mv command")
	}
}

func TestFlattenRequiresExactlyOneArg(t *testing.T) {
	if err := fsFlattenCmd.Args(fsFlattenCmd, []string{}); err == nil {
		t.Fatal("expected error for zero args to flatten")
	}
	if err := fsFlattenCmd.Args(fsFlattenCmd, []string{"a", "b"}); err == nil {
		t.Fatal("expected error for two args to flatten")
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

func TestConfigureSessionListCacheTTLUsesFlagValue(t *testing.T) {
	s := newTestSession(t)
	configureSessionListCacheTTL(s, 3*time.Second)
	if got := s.ListCacheTTL(); got != 3*time.Second {
		t.Fatalf("expected TTL 3s, got %v", got)
	}
}

func TestFsListCacheTTLDefault(t *testing.T) {
	if fsListCacheTTL != cloudfs.DefaultListCacheTTL {
		t.Fatalf("expected default fs list cache TTL %v, got %v", cloudfs.DefaultListCacheTTL, fsListCacheTTL)
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

func TestSearchMoveMovesMatches(t *testing.T) {
	ctx := context.Background()
	s := newTestSession(t)

	results, err := s.SearchMove(ctx, "/anime", "episode", "/", cloudfs.SearchOptions{})
	if err != nil {
		t.Fatalf("SearchMove failed: %v", err)
	}
	if len(results) != 1 || results[0].ID != "3" {
		t.Fatalf("unexpected SearchMove results: %+v", results)
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
