package cloudfs

import (
	"context"
	"errors"
	"path"
	"strings"
	"time"
)

const DefaultListCacheTTL = 5 * time.Second

type listCacheEntry struct {
	entries   []Entry
	expiresAt time.Time
}

type Session struct {
	driver       Driver
	root         Entry
	cwd          Entry
	cwdPath      string
	listCache    map[string]listCacheEntry
	listCacheTTL time.Duration
	now          func() time.Time
}

func NewSession(ctx context.Context, driver Driver) (*Session, error) {
	if driver == nil {
		return nil, errors.New("cloudfs: nil driver")
	}
	root, err := driver.Root(ctx)
	if err != nil {
		return nil, err
	}
	return &Session{
		driver:       driver,
		root:         root,
		cwd:          root,
		cwdPath:      "/",
		listCache:    make(map[string]listCacheEntry),
		listCacheTTL: DefaultListCacheTTL,
		now:          time.Now,
	}, nil
}

func (s *Session) Provider() string {
	return s.driver.Provider()
}

func (s *Session) Pwd() string {
	return s.cwdPath
}

func (s *Session) Cwd() Entry {
	return s.cwd
}

func (s *Session) ListCacheTTL() time.Duration {
	return s.listCacheTTL
}

func (s *Session) SetListCacheTTL(ttl time.Duration) {
	if ttl < 0 {
		ttl = 0
	}
	s.listCacheTTL = ttl
	s.Refresh()
}

func (s *Session) Refresh() {
	s.listCache = make(map[string]listCacheEntry)
}

func (s *Session) Resolve(ctx context.Context, target string) (Entry, string, error) {
	absPath := s.absPath(target)
	if absPath == "/" {
		return s.root, absPath, nil
	}

	segments := splitAbsPath(absPath)
	current := s.root
	for _, segment := range segments {
		// Resolve path segments from cached directory listings when available so
		// interactive shell navigation and completion do not re-list the same dir.
		if !current.IsDir() {
			return Entry{}, "", ErrNotDirectory
		}
		next, err := s.lookupChild(ctx, current, segment)
		if err != nil {
			return Entry{}, "", err
		}
		current = next
	}
	return current, absPath, nil
}

func (s *Session) Cd(ctx context.Context, target string) (Entry, error) {
	entry, absPath, err := s.Resolve(ctx, target)
	if err != nil {
		return Entry{}, err
	}
	if !entry.IsDir() {
		return Entry{}, ErrNotDirectory
	}
	s.cwd = entry
	s.cwdPath = absPath
	return entry, nil
}

func (s *Session) Stat(ctx context.Context, target string) (Entry, error) {
	entry, _, err := s.Resolve(ctx, target)
	return entry, err
}

func (s *Session) Ls(ctx context.Context, target string) ([]Entry, error) {
	entry, _, err := s.Resolve(ctx, target)
	if err != nil {
		return nil, err
	}
	if !entry.IsDir() {
		return []Entry{entry}, nil
	}
	return s.listDirCached(ctx, entry)
}

func (s *Session) Mkdir(ctx context.Context, target string) (Entry, error) {
	// Validate the raw basename before path.Clean swallows special segments.
	// We check the raw segment (without trimming trailing slashes) to catch "."
	// and ".." which Clean would normalise away.
	// A trailing slash is treated as a path normalisation hint (equivalent to
	// removing it), so "/anime/newdir/" is valid and equivalent to "/anime/newdir".
	rawName := rawBaseName(target)
	if rawName == "." || rawName == ".." {
		return Entry{}, ErrInvalidName
	}
	absPath := s.absPath(target)
	parentPath, name := splitParent(absPath)
	if err := validateBaseName(name); err != nil {
		return Entry{}, err
	}

	parent, _, err := s.Resolve(ctx, parentPath)
	if err != nil {
		return Entry{}, err
	}
	if !parent.IsDir() {
		return Entry{}, ErrNotDirectory
	}
	entry, err := s.driver.Mkdir(ctx, parent.ID, name)
	if err != nil {
		return Entry{}, err
	}
	s.invalidateCachedDirs(parent.ID)
	return entry, nil
}

func (s *Session) Rename(ctx context.Context, target, newName string) (Entry, error) {
	if err := validateBaseName(newName); err != nil {
		return Entry{}, err
	}
	entry, entryPath, err := s.Resolve(ctx, target)
	if err != nil {
		return Entry{}, err
	}
	result, err := s.driver.Rename(ctx, entry.ID, newName)
	if err != nil {
		return Entry{}, err
	}
	s.invalidateCachedDirs(entry.ParentID)
	// If cwd or any ancestor was renamed, refresh the session path.
	s.refreshCwdAfterRename(entryPath, newName)
	return result, nil
}

func (s *Session) Mv(ctx context.Context, targetDir string, sources ...string) ([]Entry, error) {
	target, err := s.resolveTargetDir(ctx, targetDir)
	if err != nil {
		return nil, err
	}

	results := make([]Entry, 0, len(sources))
	affectedDirIDs := make(map[string]struct{})
	defer func() {
		if len(affectedDirIDs) > 0 {
			s.invalidateCachedDirSet(affectedDirIDs)
		}
	}()
	for _, src := range sources {
		srcEntry, srcPath, err := s.Resolve(ctx, src)
		if err != nil {
			return results, err
		}
		moved, err := s.driver.Move(ctx, target.ID, srcEntry.ID)
		if err != nil {
			return results, err
		}
		affectedDirIDs[srcEntry.ParentID] = struct{}{}
		affectedDirIDs[target.ID] = struct{}{}
		results = append(results, moved)
		// If cwd or an ancestor was moved, invalidate to root.
		s.invalidateCwdIfAffected(srcPath)
	}
	return results, nil
}

func (s *Session) Cp(ctx context.Context, targetDir string, sources ...string) error {
	target, err := s.resolveTargetDir(ctx, targetDir)
	if err != nil {
		return err
	}

	affectedDirIDs := make(map[string]struct{})
	defer func() {
		if len(affectedDirIDs) > 0 {
			s.invalidateCachedDirSet(affectedDirIDs)
		}
	}()
	for _, src := range sources {
		srcEntry, _, err := s.Resolve(ctx, src)
		if err != nil {
			return err
		}
		if err := s.driver.Copy(ctx, target.ID, srcEntry.ID); err != nil {
			return err
		}
		affectedDirIDs[target.ID] = struct{}{}
	}
	return nil
}

func (s *Session) Rm(ctx context.Context, targets ...string) error {
	affectedDirIDs := make(map[string]struct{})
	defer func() {
		if len(affectedDirIDs) > 0 {
			s.invalidateCachedDirSet(affectedDirIDs)
		}
	}()
	for _, target := range targets {
		entry, entryPath, err := s.Resolve(ctx, target)
		if err != nil {
			return err
		}
		if err := s.driver.Delete(ctx, entry.ID); err != nil {
			return err
		}
		affectedDirIDs[entry.ParentID] = struct{}{}
		affectedDirIDs[entry.ID] = struct{}{}
		// If cwd or an ancestor was deleted, fall back to root.
		s.invalidateCwdIfAffected(entryPath)
	}
	return nil
}

// resolveTargetDir resolves a path and ensures it is a directory.
func (s *Session) resolveTargetDir(ctx context.Context, target string) (Entry, error) {
	entry, _, err := s.Resolve(ctx, target)
	if err != nil {
		return Entry{}, err
	}
	if !entry.IsDir() {
		return Entry{}, ErrNotDirectory
	}
	return entry, nil
}

// refreshCwdAfterRename updates cwdPath (and cwd.Name if needed) when the
// renamed entry is cwd itself or an ancestor of cwd.
func (s *Session) refreshCwdAfterRename(renamedPath, newName string) {
	if renamedPath == "/" {
		return
	}
	isCwdItself := s.cwdPath == renamedPath
	isAncestor := strings.HasPrefix(s.cwdPath, renamedPath+"/")
	if !isCwdItself && !isAncestor {
		return
	}

	parentPath, _ := splitParent(renamedPath)
	var newBase string
	if parentPath == "/" {
		newBase = "/" + newName
	} else {
		newBase = parentPath + "/" + newName
	}
	s.cwdPath = newBase + strings.TrimPrefix(s.cwdPath, renamedPath)

	// Sync cwd.Name only when cwd itself was renamed (check before cwdPath update).
	if isCwdItself {
		s.cwd.Name = newName
	}
}

// invalidateCwdIfAffected resets cwd to root when the affected path is cwd or an ancestor.
func (s *Session) invalidateCwdIfAffected(affectedPath string) {
	if affectedPath == "/" {
		return
	}
	if s.cwdPath == affectedPath || strings.HasPrefix(s.cwdPath, affectedPath+"/") {
		s.cwd = s.root
		s.cwdPath = "/"
	}
}

// validateBaseName checks that name is a valid single-segment basename.
func validateBaseName(name string) error {
	if name == "" || name == "." || name == ".." || strings.Contains(name, "/") {
		return ErrInvalidName
	}
	return nil
}

func (s *Session) lookupChild(ctx context.Context, parent Entry, name string) (Entry, error) {
	entries, err := s.listDirCached(ctx, parent)
	if err != nil {
		return Entry{}, err
	}
	var matched *Entry
	for _, entry := range entries {
		if entry.Name != name {
			continue
		}
		if matched != nil {
			return Entry{}, ErrAmbiguousPath
		}
		copy := entry
		matched = &copy
	}
	if matched == nil {
		return Entry{}, ErrNotFound
	}
	return *matched, nil
}

func (s *Session) listDirCached(ctx context.Context, dir Entry) ([]Entry, error) {
	if !dir.IsDir() {
		return nil, ErrNotDirectory
	}
	if cached, ok := s.listCache[dir.ID]; ok && s.cacheEntryValid(cached) {
		return cloneEntries(cached.entries), nil
	}
	entries, err := s.driver.List(ctx, dir.ID)
	if err != nil {
		return nil, err
	}
	if s.listCacheTTL > 0 {
		now := s.now()
		s.listCache[dir.ID] = listCacheEntry{
			entries:   cloneEntries(entries),
			expiresAt: now.Add(s.listCacheTTL),
		}
	}
	return cloneEntries(entries), nil
}

func (s *Session) cacheEntryValid(entry listCacheEntry) bool {
	if s.listCacheTTL <= 0 {
		return false
	}
	return s.now().Before(entry.expiresAt)
}

func (s *Session) invalidateCachedDirs(dirIDs ...string) {
	for _, dirID := range dirIDs {
		if dirID == "" {
			continue
		}
		delete(s.listCache, dirID)
	}
}

func (s *Session) invalidateCachedDirSet(dirIDs map[string]struct{}) {
	for dirID := range dirIDs {
		if dirID == "" {
			continue
		}
		delete(s.listCache, dirID)
	}
}

func cloneEntries(entries []Entry) []Entry {
	cloned := make([]Entry, len(entries))
	copy(cloned, entries)
	return cloned
}

func (s *Session) absPath(input string) string {
	if input == "" {
		return s.cwdPath
	}
	if strings.HasPrefix(input, "/") {
		return cleanPath(input)
	}
	return cleanPath(path.Join(s.cwdPath, input))
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	p = path.Clean("/" + strings.TrimPrefix(p, "/"))
	if p == "." {
		return "/"
	}
	return p
}

func splitAbsPath(absPath string) []string {
	if absPath == "/" {
		return nil
	}
	return strings.Split(strings.Trim(absPath, "/"), "/")
}

func splitParent(absPath string) (string, string) {
	parent := path.Dir(absPath)
	if parent == "." {
		parent = "/"
	}
	return cleanPath(parent), path.Base(absPath)
}

// rawBaseName extracts the last path segment from the raw (uncleaned) input,
// without trimming trailing slashes. Used to detect "", ".", ".." before
// path.Clean normalises them away.
func rawBaseName(p string) string {
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return p
	}
	return p[idx+1:]
}
