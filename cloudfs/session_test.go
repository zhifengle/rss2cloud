package cloudfs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeDriver is an in-memory Driver for testing Session semantics.
type fakeDriver struct {
	entries  map[string]Entry
	children map[string][]string
	nextID   int
}

type countingFakeDriver struct {
	*fakeDriver
	listCalls map[string]int
}

func newFakeDriver() *fakeDriver {
	return &fakeDriver{
		entries: map[string]Entry{
			"0": {ID: "0", Name: "/", Type: EntryTypeDirectory},
			"1": {ID: "1", ParentID: "0", Name: "anime", Type: EntryTypeDirectory},
			"2": {ID: "2", ParentID: "0", Name: "notes.txt", Type: EntryTypeFile},
			"3": {ID: "3", ParentID: "1", Name: "sub", Type: EntryTypeDirectory},
			"4": {ID: "4", ParentID: "1", Name: "episode.mkv", Type: EntryTypeFile},
			"5": {ID: "5", ParentID: "3", Name: "episode-02.mkv", Type: EntryTypeFile},
		},
		children: map[string][]string{
			"0": {"1", "2"},
			"1": {"3", "4"},
			"3": {"5"},
		},
		nextID: 6,
	}
}

func newCountingFakeDriver() *countingFakeDriver {
	return &countingFakeDriver{
		fakeDriver: newFakeDriver(),
		listCalls:  make(map[string]int),
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

func (d *countingFakeDriver) List(ctx context.Context, dirID string) ([]Entry, error) {
	d.listCalls[dirID]++
	return d.fakeDriver.List(ctx, dirID)
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

func (d *fakeDriver) Search(_ context.Context, dirID, keyword string, opts SearchOptions) ([]Entry, error) {
	var results []Entry
	var visit func(string)
	visit = func(currentID string) {
		for _, childID := range d.children[currentID] {
			entry := d.entries[childID]
			if entry.IsDir() {
				if opts.IncludeDirectories && contains(entry.Name, keyword) {
					results = append(results, entry)
				}
				visit(childID)
				continue
			}
			if contains(entry.Name, keyword) {
				results = append(results, entry)
			}
		}
	}
	visit(dirID)
	return results, nil
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

func contains(name, keyword string) bool {
	return keyword == "" || (keyword != "" && strings.Contains(name, keyword))
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

func TestSearchReturnsRecursiveFileMatches(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	results, err := s.Search(ctx, "/anime", "episode", SearchOptions{})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %+v", results)
	}
	if results[0].IsDir() || results[1].IsDir() {
		t.Fatalf("expected file-only results, got %+v", results)
	}
}

func TestSearchMoveMovesMatchesIntoTargetDir(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	results, err := s.SearchMove(ctx, "/anime", "episode", "/", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchMove failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 moved entries, got %+v", results)
	}
	for _, entry := range results {
		if entry.ParentID != "0" {
			t.Fatalf("expected moved entry under root, got %+v", entry)
		}
	}
}

func TestSearchMoveRejectsDuplicateTargetNames(t *testing.T) {
	ctx := context.Background()
	d := newFakeDriver()
	d.entries["6"] = Entry{ID: "6", ParentID: "0", Name: "episode.mkv", Type: EntryTypeFile}
	d.children["0"] = append(d.children["0"], "6")

	s, _ := NewSession(ctx, d)
	results, err := s.SearchMove(ctx, "/anime", "episode", "/", SearchOptions{})
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no moved results on conflict, got %+v", results)
	}
	entry, statErr := s.Stat(ctx, "/anime/episode.mkv")
	if statErr != nil || entry.ParentID != "1" {
		t.Fatalf("expected original file to stay put, got entry=%+v err=%v", entry, statErr)
	}
}

func TestFlattenRequiresDirectory(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	_, err := s.Flatten(ctx, "/notes.txt", FlattenOptions{})
	if !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("expected ErrNotDirectory, got %v", err)
	}
}

func TestFlattenReturnsResolvedTargetAndNoError(t *testing.T) {
	ctx := context.Background()
	s, _ := NewSession(ctx, newFakeDriver())

	result, err := s.Flatten(ctx, "/anime", FlattenOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Target.ID != "1" {
		t.Fatalf("expected resolved target to be anime, got %+v", result.Target)
	}
	if len(result.Moved) != 1 || result.Moved[0].ID != "5" {
		t.Fatalf("expected moved nested file episode-02.mkv, got %+v", result.Moved)
	}
	if len(result.RemovedDirs) != 1 || result.RemovedDirs[0].ID != "3" {
		t.Fatalf("expected removed descendant dir sub, got %+v", result.RemovedDirs)
	}
}

func TestFlattenDoesNotRemoveOriginallyEmptyDescendantDirs(t *testing.T) {
	ctx := context.Background()
	d := newFakeDriver()
	d.entries["6"] = Entry{ID: "6", ParentID: "1", Name: "empty", Type: EntryTypeDirectory}
	d.children["1"] = append(d.children["1"], "6")
	d.children["6"] = []string{}

	s, _ := NewSession(ctx, d)
	result, err := s.Flatten(ctx, "/anime", FlattenOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.RemovedDirs) != 1 || result.RemovedDirs[0].ID != "3" {
		t.Fatalf("expected only descendant dir with moved files to be removed, got %+v", result.RemovedDirs)
	}
	if _, _, err := s.Resolve(ctx, "/anime/empty"); err != nil {
		t.Fatalf("expected originally empty dir to remain, got %v", err)
	}
}

func TestFlattenDryRunDoesNotPlanRemovalForOriginallyEmptyDirs(t *testing.T) {
	ctx := context.Background()
	d := newFakeDriver()
	d.entries["6"] = Entry{ID: "6", ParentID: "1", Name: "empty", Type: EntryTypeDirectory}
	d.children["1"] = append(d.children["1"], "6")
	d.children["6"] = []string{}

	s, _ := NewSession(ctx, d)
	result, err := s.Flatten(ctx, "/anime", FlattenOptions{DryRun: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.PlannedRemovals) != 1 || result.PlannedRemovals[0].ID != "3" {
		t.Fatalf("expected only descendant dir with planned file moves to be removed, got %+v", result.PlannedRemovals)
	}
	if len(result.RemovedDirs) != 0 {
		t.Fatalf("expected no removals in dry-run, got %+v", result.RemovedDirs)
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

func TestMkdirInvalidatesOnlyParentDirectoryCache(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	s, _ := NewSession(ctx, d)

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime"); err != nil {
		t.Fatalf("Ls(/anime) failed: %v", err)
	}
	if _, err := s.Mkdir(ctx, "/anime/newdir"); err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	if _, err := s.Ls(ctx, "/anime"); err != nil {
		t.Fatalf("Ls(/anime) after mkdir failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) after mkdir failed: %v", err)
	}

	if got := d.listCalls["1"]; got != 2 {
		t.Fatalf("expected /anime list reloaded once, got %d", got)
	}
	if got := d.listCalls["0"]; got != 1 {
		t.Fatalf("expected root cache to stay warm, got %d", got)
	}
}

func TestMvInvalidatesSourceAndTargetDirectoryCachesOnly(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	s, _ := NewSession(ctx, d)

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime"); err != nil {
		t.Fatalf("Ls(/anime) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime/sub"); err != nil {
		t.Fatalf("Ls(/anime/sub) failed: %v", err)
	}

	if _, err := s.Mv(ctx, "/anime/sub", "/notes.txt"); err != nil {
		t.Fatalf("Mv failed: %v", err)
	}

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) after mv failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime/sub"); err != nil {
		t.Fatalf("Ls(/anime/sub) after mv failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime"); err != nil {
		t.Fatalf("Ls(/anime) after mv failed: %v", err)
	}

	if got := d.listCalls["0"]; got != 2 {
		t.Fatalf("expected root list to reload after mv, got %d", got)
	}
	if got := d.listCalls["3"]; got != 2 {
		t.Fatalf("expected target dir list to reload after mv, got %d", got)
	}
	if got := d.listCalls["1"]; got != 1 {
		t.Fatalf("expected unrelated /anime cache to stay warm, got %d", got)
	}
}

func TestSearchMoveInvalidatesOnlyMovedDirectories(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	s, _ := NewSession(ctx, d)

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime"); err != nil {
		t.Fatalf("Ls(/anime) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime/sub"); err != nil {
		t.Fatalf("Ls(/anime/sub) failed: %v", err)
	}
	rootBefore := d.listCalls["0"]
	animeBefore := d.listCalls["1"]
	subBefore := d.listCalls["3"]

	if _, err := s.SearchMove(ctx, "/anime", "episode", "/", SearchOptions{}); err != nil {
		t.Fatalf("SearchMove failed: %v", err)
	}

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) after search-move failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime"); err != nil {
		t.Fatalf("Ls(/anime) after search-move failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime/sub"); err != nil {
		t.Fatalf("Ls(/anime/sub) after search-move failed: %v", err)
	}

	if got := d.listCalls["0"] - rootBefore; got != 2 {
		t.Fatalf("expected root list to be read once for planning and once after invalidation, got delta %d", got)
	}
	if got := d.listCalls["1"] - animeBefore; got != 1 {
		t.Fatalf("expected /anime list to reload once after invalidation, got delta %d", got)
	}
	if got := d.listCalls["3"] - subBefore; got != 1 {
		t.Fatalf("expected /anime/sub list to reload once after invalidation, got delta %d", got)
	}
}

func TestFlattenKeepsUnrelatedDirectoryCacheWarm(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	d.entries["6"] = Entry{ID: "6", ParentID: "0", Name: "docs", Type: EntryTypeDirectory}
	d.children["0"] = append(d.children["0"], "6")
	d.children["6"] = []string{}
	s, _ := NewSession(ctx, d)

	if _, err := s.Ls(ctx, "/docs"); err != nil {
		t.Fatalf("Ls(/docs) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime"); err != nil {
		t.Fatalf("Ls(/anime) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime/sub"); err != nil {
		t.Fatalf("Ls(/anime/sub) failed: %v", err)
	}
	docsBefore := d.listCalls["6"]
	animeBefore := d.listCalls["1"]

	if _, err := s.Flatten(ctx, "/anime", FlattenOptions{}); err != nil {
		t.Fatalf("Flatten failed: %v", err)
	}

	if _, err := s.Ls(ctx, "/docs"); err != nil {
		t.Fatalf("Ls(/docs) after flatten failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/anime"); err != nil {
		t.Fatalf("Ls(/anime) after flatten failed: %v", err)
	}

	if got := d.listCalls["6"] - docsBefore; got != 0 {
		t.Fatalf("expected unrelated /docs cache to stay warm, got delta %d", got)
	}
	if got := d.listCalls["1"] - animeBefore; got != 2 {
		t.Fatalf("expected /anime to be read once for planning and once after invalidation, got delta %d", got)
	}
}

func TestListCacheTTLUsesCachedEntryBeforeExpiry(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	s, _ := NewSession(ctx, d)
	base := time.Unix(100, 0)
	s.listCacheTTL = 5 * time.Second
	s.now = func() time.Time { return base }

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("second Ls(/) failed: %v", err)
	}

	if got := d.listCalls["0"]; got != 1 {
		t.Fatalf("expected cached root listing before TTL expiry, got %d calls", got)
	}
}

func TestListCacheTTLReloadsAfterExpiry(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	s, _ := NewSession(ctx, d)
	current := time.Unix(100, 0)
	s.listCacheTTL = 5 * time.Second
	s.now = func() time.Time { return current }

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	current = current.Add(6 * time.Second)
	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) after expiry failed: %v", err)
	}

	if got := d.listCalls["0"]; got != 2 {
		t.Fatalf("expected root listing to reload after TTL expiry, got %d calls", got)
	}
}

func TestSessionRefreshClearsCachedListings(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	s, _ := NewSession(ctx, d)
	s.listCacheTTL = time.Minute

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	s.Refresh()
	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) after refresh failed: %v", err)
	}

	if got := d.listCalls["0"]; got != 2 {
		t.Fatalf("expected refresh to clear root cache, got %d calls", got)
	}
}

func TestSetListCacheTTLDisablesCaching(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	s, _ := NewSession(ctx, d)
	s.SetListCacheTTL(0)

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("second Ls(/) failed: %v", err)
	}

	if got := s.ListCacheTTL(); got != 0 {
		t.Fatalf("expected TTL 0 after disabling cache, got %v", got)
	}
	if got := d.listCalls["0"]; got != 2 {
		t.Fatalf("expected disabled cache to reload every time, got %d calls", got)
	}
}

func TestSetListCacheTTLRefreshesExistingCache(t *testing.T) {
	ctx := context.Background()
	d := newCountingFakeDriver()
	s, _ := NewSession(ctx, d)
	s.SetListCacheTTL(time.Minute)

	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) failed: %v", err)
	}
	s.SetListCacheTTL(2 * time.Second)
	if _, err := s.Ls(ctx, "/"); err != nil {
		t.Fatalf("Ls(/) after TTL change failed: %v", err)
	}

	if got := s.ListCacheTTL(); got != 2*time.Second {
		t.Fatalf("expected updated TTL, got %v", got)
	}
	if got := d.listCalls["0"]; got != 2 {
		t.Fatalf("expected TTL change to flush cached root listing, got %d calls", got)
	}
}
