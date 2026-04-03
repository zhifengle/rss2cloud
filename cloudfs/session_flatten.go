package cloudfs

import (
	"context"
	"path"
	"sort"
)

// FlattenOptions controls future flatten execution behaviour.
type FlattenOptions struct {
	DryRun        bool
	KeepEmptyDirs bool
}

// FlattenMove describes one future move from a nested path into the target dir.
type FlattenMove struct {
	Source Entry
}

// FlattenResult is the long-term result model for the flatten composite command.
type FlattenResult struct {
	Target          Entry
	PlannedMoves    []FlattenMove
	PlannedRemovals []Entry
	Moved           []Entry
	RemovedDirs     []Entry
}

type flattenDirPlan struct {
	entry    Entry
	path     string
	hadFiles bool
}

type flattenPlan struct {
	files []Entry
	dirs  []flattenDirPlan
}

// Flatten expands descendant files into the target directory and optionally
// removes emptied descendant directories.
func (s *Session) Flatten(ctx context.Context, target string, opts FlattenOptions) (FlattenResult, error) {
	result := FlattenResult{}

	targetDir, targetPath, err := s.Resolve(ctx, target)
	if err != nil {
		return result, err
	}
	if !targetDir.IsDir() {
		return result, ErrNotDirectory
	}
	result.Target = targetDir

	targetEntries, err := s.driver.List(ctx, targetDir.ID)
	if err != nil {
		return result, err
	}
	sort.Slice(targetEntries, func(i, j int) bool {
		return targetEntries[i].Name < targetEntries[j].Name
	})

	plan := flattenPlan{}
	for _, entry := range targetEntries {
		if !entry.IsDir() {
			continue
		}
		subPlan, err := s.buildFlattenPlan(ctx, entry, path.Join(targetPath, entry.Name))
		if err != nil {
			return result, err
		}
		plan.files = append(plan.files, subPlan.files...)
		plan.dirs = append(plan.dirs, subPlan.dirs...)
	}

	result.PlannedMoves = make([]FlattenMove, 0, len(plan.files))
	for _, file := range plan.files {
		result.PlannedMoves = append(result.PlannedMoves, FlattenMove{Source: file})
	}
	if !opts.KeepEmptyDirs {
		result.PlannedRemovals = make([]Entry, 0, len(plan.dirs))
		for _, dir := range plan.dirs {
			if !dir.hadFiles {
				continue
			}
			result.PlannedRemovals = append(result.PlannedRemovals, dir.entry)
		}
	}

	if err := checkFlattenConflicts(targetEntries, plan.files); err != nil {
		return result, err
	}
	if opts.DryRun {
		return result, nil
	}

	result.Moved = make([]Entry, 0, len(plan.files))
	affectedDirIDs := map[string]struct{}{
		targetDir.ID: {},
	}
	for _, file := range plan.files {
		moved, err := s.driver.Move(ctx, targetDir.ID, file.ID)
		if err != nil {
			return result, err
		}
		affectedDirIDs[file.ParentID] = struct{}{}
		result.Moved = append(result.Moved, moved)
	}

	if opts.KeepEmptyDirs {
		return result, nil
	}

	result.RemovedDirs = make([]Entry, 0, len(plan.dirs))
	for _, dir := range plan.dirs {
		if !dir.hadFiles {
			continue
		}
		entries, err := s.driver.List(ctx, dir.entry.ID)
		if err != nil {
			return result, err
		}
		if len(entries) != 0 {
			continue
		}
		if err := s.driver.Delete(ctx, dir.entry.ID); err != nil {
			return result, err
		}
		affectedDirIDs[dir.entry.ParentID] = struct{}{}
		affectedDirIDs[dir.entry.ID] = struct{}{}
		result.RemovedDirs = append(result.RemovedDirs, dir.entry)
		s.invalidateCwdIfAffected(dir.path)
	}

	if len(result.Moved) > 0 || len(result.RemovedDirs) > 0 {
		s.invalidateCachedDirSet(affectedDirIDs)
	}

	return result, nil
}

func (s *Session) buildFlattenPlan(ctx context.Context, dir Entry, dirPath string) (flattenPlan, error) {
	entries, err := s.driver.List(ctx, dir.ID)
	if err != nil {
		return flattenPlan{}, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	plan := flattenPlan{}
	hadFiles := false
	for _, entry := range entries {
		if entry.IsDir() {
			subPlan, err := s.buildFlattenPlan(ctx, entry, path.Join(dirPath, entry.Name))
			if err != nil {
				return flattenPlan{}, err
			}
			plan.files = append(plan.files, subPlan.files...)
			plan.dirs = append(plan.dirs, subPlan.dirs...)
			continue
		}
		hadFiles = true
		plan.files = append(plan.files, entry)
	}

	plan.dirs = append(plan.dirs, flattenDirPlan{entry: dir, path: dirPath, hadFiles: hadFiles})
	return plan, nil
}

func checkFlattenConflicts(targetEntries, plannedFiles []Entry) error {
	reservedNames := make(map[string]struct{}, len(targetEntries)+len(plannedFiles))
	for _, entry := range targetEntries {
		reservedNames[entry.Name] = struct{}{}
	}
	for _, entry := range plannedFiles {
		if _, ok := reservedNames[entry.Name]; ok {
			return ErrAlreadyExists
		}
		reservedNames[entry.Name] = struct{}{}
	}
	return nil
}
