package cloudfs

import (
	"context"
	"errors"
	"path"
	"strings"
)

type Session struct {
	driver  Driver
	root    Entry
	cwd     Entry
	cwdPath string
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
		driver:  driver,
		root:    root,
		cwd:     root,
		cwdPath: "/",
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

func (s *Session) Resolve(ctx context.Context, target string) (Entry, string, error) {
	absPath := s.absPath(target)
	if absPath == "/" {
		return s.root, absPath, nil
	}

	segments := splitAbsPath(absPath)
	current := s.root
	for _, segment := range segments {
		// Each Lookup requires the current node to be a directory.
		if !current.IsDir() {
			return Entry{}, "", ErrNotDirectory
		}
		next, err := s.driver.Lookup(ctx, current.ID, segment)
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
	return s.driver.List(ctx, entry.ID)
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
	return s.driver.Mkdir(ctx, parent.ID, name)
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
	for _, src := range sources {
		srcEntry, srcPath, err := s.Resolve(ctx, src)
		if err != nil {
			return results, err
		}
		moved, err := s.driver.Move(ctx, target.ID, srcEntry.ID)
		if err != nil {
			return results, err
		}
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

	for _, src := range sources {
		srcEntry, _, err := s.Resolve(ctx, src)
		if err != nil {
			return err
		}
		if err := s.driver.Copy(ctx, target.ID, srcEntry.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) Rm(ctx context.Context, targets ...string) error {
	for _, target := range targets {
		entry, entryPath, err := s.Resolve(ctx, target)
		if err != nil {
			return err
		}
		if err := s.driver.Delete(ctx, entry.ID); err != nil {
			return err
		}
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
