package cloudfs

import "context"

// Search resolves searchRoot as a directory and returns provider search results.
func (s *Session) Search(ctx context.Context, searchRoot, keyword string, opts SearchOptions) ([]Entry, error) {
	root, err := s.resolveTargetDir(ctx, searchRoot)
	if err != nil {
		return nil, err
	}

	searcher, ok := s.driver.(Searcher)
	if !ok {
		return nil, ErrUnsupported
	}
	return searcher.Search(ctx, root.ID, keyword, opts)
}

// SearchMove searches files under searchRoot and moves matched files into targetDir.
// It pre-checks basename conflicts in targetDir before applying moves.
func (s *Session) SearchMove(
	ctx context.Context,
	searchRoot, keyword, targetDir string,
	opts SearchOptions,
) ([]Entry, error) {
	target, err := s.resolveTargetDir(ctx, targetDir)
	if err != nil {
		return nil, err
	}

	matches, err := s.Search(ctx, searchRoot, keyword, opts)
	if err != nil {
		return nil, err
	}

	planned := make([]Entry, 0, len(matches))
	seenIDs := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if match.IsDir() {
			continue
		}
		if match.ParentID == target.ID {
			continue
		}
		if _, ok := seenIDs[match.ID]; ok {
			continue
		}
		seenIDs[match.ID] = struct{}{}
		planned = append(planned, match)
	}

	if len(planned) == 0 {
		return nil, nil
	}

	targetEntries, err := s.driver.List(ctx, target.ID)
	if err != nil {
		return nil, err
	}
	reservedNames := make(map[string]struct{}, len(targetEntries)+len(planned))
	for _, entry := range targetEntries {
		reservedNames[entry.Name] = struct{}{}
	}
	for _, entry := range planned {
		if _, ok := reservedNames[entry.Name]; ok {
			return nil, ErrAlreadyExists
		}
		reservedNames[entry.Name] = struct{}{}
	}

	results := make([]Entry, 0, len(planned))
	for _, entry := range planned {
		moved, err := s.driver.Move(ctx, target.ID, entry.ID)
		if err != nil {
			return results, err
		}
		results = append(results, moved)
	}
	return results, nil
}
