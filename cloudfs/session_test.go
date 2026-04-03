package cloudfs

import (
	"context"
	"errors"
	"testing"
)

// fakeDriver is an in-memory Driver for testing Session semantics.
type fakeDriver struct {
	entries  map[string]Entry
	children map[string][]string
	nextID   int
}

func newFakeDriver() *fakeDriver {
	return &fakeDriver{
		entries: map[string]Entry{
			"0": {ID: "0", Name: "/", Type: EntryTypeDirectory},
			"1": {ID: "1", ParentID: "0", Name: "anime", Type: EntryTypeDirectory},
			"2": {ID: "2", ParentID: "0", Name: "notes.txt", Type: EntryTypeFile},
			"3": {ID: "3", ParentID: "1", Name: "sub", Type: EntryTypeDirectory},
			"4": {ID: "4", ParentID: "1", Name: "episode.mkv", Type: EntryTypeFile},
		},
		children: map[string][]string{
			"0": {"1", "2"},
			"1": {"3", "4"},
			"3": {},
		},
		nextID: 5,
	}
}

func (d *fakeDriver) Provider() string { return "fake" }

func (d *fakeDriver) Root(_ context.Context) (Entry, error) {
	return d.entries["0"], nil
}

func (d *fakeDriver) Stat(_ context.Context, entryID string) (Entry, error) {
	e, ok := d.entries[entryID]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return e, nil
}

func (d *fakeDriver) List(_ context.Context, dirID string) ([]Entry, error) {
	if dirID == "" {
		dirID = "0"
	}
	childIDs, ok := d.children[dirID]
	if !ok {
		return nil, ErrNotFound
	}
	items := make([]Entry, 0, len(childIDs))
	for _, id := range childIDs {
		items = append(items, d.entries[id])
	}
	return items, nil
}

func (d *fakeDriver) Lookup(_ context.Context, parentID, name string) (Entry, error) {
	var matched *Entry
	for _, id := range d.children[parentID] {
		e := d.entries[id]
		if e.Name == name {
			if matched != nil {
				return Entry{}, ErrAmbiguousPath
			}
			copy := e
			matched = &copy
		}
	}
	if matched == nil {
		return Entry{}, ErrNotFound
	}
	return *matched, nil
}

func (d *fakeDriver) Mkdir(_ context.Context, parentID, name string) (Entry, error) {
	id := string(rune('0' + d.nextID))
	d.nextID++
	e := Entry{ID: id, ParentID: parentID, Name: name, Type: EntryTypeDirectory}
	d.entries[id] = e
	d.children[parentID] = append(d.children[parentID], id)
	d.children[id] = []string{}
	return e, nil
}

func (d *fakeDriver) Rename(_ context.Context, entryID, newName string) (Entry, error) {
	e, ok := d.entries[entryID]
	if !ok {
		return Entry{}, ErrNotFound
	}
	e.Name = newName
	d.entries[entryID] = e
	return e, nil
}

func (d *fakeDriver) Move(_ context.Context, targetDirID, entryID string) (Entry, error) {
	e, ok := d.entries[entryID]
	if !ok {
		return Entry{}, ErrNotFound
	}
	// update parent in children maps
	oldParent := e.ParentID
	d.children[oldParent] = removeID(d.children[oldParent], entryID)
	d.children[targetDirID] = append(d.children[targetDirID], entryID)
	e.ParentID = targetDirID
	d.entries[entryID] = e
	return e, nil
}

func (d *fakeDriver) Copy(_ context.Context, targetDirID, entryID string) error {
	_, ok := d.entries[entryID]
	if !ok {
		return ErrNotFound
	}
	_ = targetDirID
	return nil
}

func (d *fakeDriver) Delete(_ context.Context, entryID string) error {
	e, ok := d.entries[entryID]
	if !ok {
		return ErrNotFound
	}
	delete(d.entries, entryID)
	d.children[e.ParentID] = removeID(d.children[e.ParentID], entryID)
	return nil
}

func removeID(ids []string, id string) []string {
	out := ids[:0]
	for _, v := range ids {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}

// --- Tests ---

func TestResolveRootVariants(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	for _, input := range []string{"/", "/.", "/.."} {
		e, p, err := s.Resolve(ctx, input)
		if err != nil {
			t.Fatalf("Resolve(%q) error: %v", input, err)
		}
		if p != "/" {
			t.Fatalf("Resolve(%q) path = %q, want /", input, p)
		}
		if e.ID != "0" {
			t.Fatalf("Resolve(%q) ID = %q, want 0", input, e.ID)
		}
	}
}

func TestResolveDotAndDotDot(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())
	s.Cd(ctx, "/anime/sub")

	// "." should resolve to cwd
	e, p, err := s.Resolve(ctx, ".")
	if err != nil || p != "/anime/sub" || e.ID != "3" {
		t.Fatalf("Resolve('.') = (%v, %q, %v)", e, p, err)
	}

	// ".." should resolve to parent
	e, p, err = s.Resolve(ctx, "..")
	if err != nil || p != "/anime" || e.ID != "1" {
		t.Fatalf("Resolve('..') = (%v, %q, %v)", e, p, err)
	}
}

func TestCdAndPwd(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	if _, err := s.Cd(ctx, "anime"); err != nil {
		t.Fatal(err)
	}
	if s.Pwd() != "/anime" {
		t.Fatalf("pwd = %q, want /anime", s.Pwd())
	}

	if _, err := s.Cd(ctx, "./sub"); err != nil {
		t.Fatal(err)
	}
	if s.Pwd() != "/anime/sub" {
		t.Fatalf("pwd = %q, want /anime/sub", s.Pwd())
	}
}

func TestCdOnFileReturnsError(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())
	_, err := s.Cd(ctx, "/notes.txt")
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("expected ErrNotDirectory, got %v", err)
	}
}

func TestLs(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	items, err := s.Ls(ctx, "/")
	if err != nil || len(items) != 2 {
		t.Fatalf("Ls('/') = (%v, %v)", items, err)
	}

	// Ls on a file returns the file itself
	items, err = s.Ls(ctx, "/notes.txt")
	if err != nil || len(items) != 1 || items[0].ID != "2" {
		t.Fatalf("Ls('/notes.txt') = (%v, %v)", items, err)
	}
}

func TestMkdirOnlyCreatesLastSegment(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	// /anime/b/c — parent /anime/b does not exist, should fail
	_, err := s.Mkdir(ctx, "/anime/b/c")
	if err == nil {
		t.Fatal("expected error for missing intermediate dir, got nil")
	}

	// /anime/newdir — parent /anime exists
	e, err := s.Mkdir(ctx, "/anime/newdir")
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}
	if e.Name != "newdir" || e.ParentID != "1" {
		t.Fatalf("unexpected entry: %+v", e)
	}
}

func TestMkdirInvalidNames(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	// "." and ".." as the last raw segment are rejected before path.Clean normalises them.
	for _, name := range []string{".", ".."} {
		_, err := s.Mkdir(ctx, "/anime/"+name)
		if !errors.Is(err, ErrInvalidName) {
			t.Fatalf("Mkdir with name %q: expected ErrInvalidName, got %v", name, err)
		}
	}

	// Trailing slash is a normalisation hint: "/anime/" == "/anime", not an empty name.
	// "/anime/a/b" — parent "/anime/a" doesn't exist.
	_, err := s.Mkdir(ctx, "/anime/a/b")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Mkdir /anime/a/b: expected ErrNotFound (missing parent), got %v", err)
	}
}

func TestRenameReturnsEntry(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	e, err := s.Rename(ctx, "/anime/episode.mkv", "episode-01.mkv")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	if e.ID != "4" || e.Name != "episode-01.mkv" {
		t.Fatalf("unexpected renamed entry: %+v", e)
	}
}

func TestRenameInvalidNames(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	for _, name := range []string{"", ".", "..", "a/b"} {
		_, err := s.Rename(ctx, "/anime/episode.mkv", name)
		if !errors.Is(err, ErrInvalidName) {
			t.Fatalf("Rename with newName %q: expected ErrInvalidName, got %v", name, err)
		}
	}
}

func TestMvReturnsEntries(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	results, err := s.Mv(ctx, "/anime/sub", "/notes.txt")
	if err != nil {
		t.Fatalf("Mv failed: %v", err)
	}
	if len(results) != 1 || results[0].ID != "2" {
		t.Fatalf("unexpected Mv results: %+v", results)
	}
}

func TestMvTargetNotDirectory(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	_, err := s.Mv(ctx, "/notes.txt", "/anime/episode.mkv")
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("expected ErrNotDirectory, got %v", err)
	}
}

func TestCpOnlyReturnsError(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	err := s.Cp(ctx, "/anime/sub", "/notes.txt")
	if err != nil {
		t.Fatalf("Cp failed: %v", err)
	}
}

func TestCpTargetNotDirectory(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	err := s.Cp(ctx, "/notes.txt", "/anime/episode.mkv")
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("expected ErrNotDirectory, got %v", err)
	}
}

func TestRmSerialStopsOnError(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	// Delete two real entries — should succeed
	err := s.Rm(ctx, "/notes.txt", "/anime/episode.mkv")
	if err != nil {
		t.Fatalf("Rm failed: %v", err)
	}

	// Now try to delete something that no longer exists
	err = s.Rm(ctx, "/notes.txt")
	if err == nil {
		t.Fatal("expected error deleting non-existent entry")
	}
}

func TestAmbiguousPath(t *testing.T) {
	ctx := context.Background()
	d := newFakeDriver()
	// Add a duplicate name under root
	d.entries["99"] = Entry{ID: "99", ParentID: "0", Name: "anime", Type: EntryTypeDirectory}
	d.children["0"] = append(d.children["0"], "99")

	s, _ := NewSession(ctx, d)
	_, _, err := s.Resolve(ctx, "/anime")
	if !errors.Is(err, ErrAmbiguousPath) {
		t.Fatalf("expected ErrAmbiguousPath, got %v", err)
	}
}

func TestResolveIntermediateNotDirectory(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	// /notes.txt is a file; /notes.txt/child should return ErrNotDirectory.
	_, _, err := s.Resolve(ctx, "/notes.txt/child")
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("expected ErrNotDirectory for file-as-parent, got %v", err)
	}
}

func TestMkdirTrailingSlashIsValid(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	// Trailing slash should be normalised away, not rejected.
	e, err := s.Mkdir(ctx, "/anime/newdir/")
	if err != nil {
		t.Fatalf("Mkdir with trailing slash failed: %v", err)
	}
	if e.Name != "newdir" {
		t.Fatalf("expected name 'newdir', got %q", e.Name)
	}
}

func TestRenameCwdUpdatesPath(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())
	s.Cd(ctx, "/anime")

	_, err := s.Rename(ctx, "/anime", "shows")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	if s.Pwd() != "/shows" {
		t.Fatalf("expected pwd /shows after rename, got %q", s.Pwd())
	}
	if s.Cwd().Name != "shows" {
		t.Fatalf("expected Cwd().Name 'shows' after rename, got %q", s.Cwd().Name)
	}
}

func TestRenameAncestorUpdatesCwdPath(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())
	s.Cd(ctx, "/anime/sub")

	_, err := s.Rename(ctx, "/anime", "shows")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	if s.Pwd() != "/shows/sub" {
		t.Fatalf("expected pwd /shows/sub after ancestor rename, got %q", s.Pwd())
	}
}

func TestMvCwdFallsBackToRoot(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())
	s.Cd(ctx, "/anime/sub")

	// Move /anime into root (it's already there, but the fake driver accepts it).
	// What matters is that cwd gets invalidated when its ancestor is moved.
	_, err := s.Mv(ctx, "/", "/anime")
	if err != nil {
		t.Fatalf("Mv failed: %v", err)
	}
	if s.Pwd() != "/" {
		t.Fatalf("expected pwd / after ancestor moved, got %q", s.Pwd())
	}
}

func TestRmCwdFallsBackToRoot(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())
	s.Cd(ctx, "/anime/sub")

	err := s.Rm(ctx, "/anime/sub")
	if err != nil {
		t.Fatalf("Rm failed: %v", err)
	}
	if s.Pwd() != "/" {
		t.Fatalf("expected pwd / after cwd deleted, got %q", s.Pwd())
	}
}
