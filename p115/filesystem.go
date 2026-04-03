package p115

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/deadblue/elevengo"
	llErrors "github.com/deadblue/elevengo/lowlevel/errors"
	"github.com/deadblue/elevengo/option"
	"github.com/zhifengle/rss2cloud/cloudfs"
)

// FileSystemOption holds provider-level configuration for FileSystem.
type FileSystemOption struct {
	// RootID overrides the logical root directory (default "0").
	RootID string
	// PageSize is a hint for list page size.
	// Note: elevengo does not expose per-call page size control; this field is
	// reserved for future use or a lower-level workaround.
	PageSize int
	// OperationLimiter is an optional rate limiter applied at the operation layer.
	OperationLimiter cloudfs.Limiter
}

const defaultRootID = "0"

type FileSystem struct {
	agent    *elevengo.Agent
	rootID   string
	pageSize int
	limiter  cloudfs.Limiter
}

// NewFileSystem creates a FileSystem with default options.
func NewFileSystem(agent *elevengo.Agent) *FileSystem {
	return NewFileSystemWithOption(agent, FileSystemOption{})
}

// NewFileSystemWithOption creates a FileSystem with the given options.
func NewFileSystemWithOption(agent *elevengo.Agent, opt FileSystemOption) *FileSystem {
	rootID := opt.RootID
	if rootID == "" {
		rootID = defaultRootID
	}
	return &FileSystem{
		agent:    agent,
		rootID:   rootID,
		pageSize: opt.PageSize,
		limiter:  opt.OperationLimiter,
	}
}

// FileSystem returns a cloudfs.Driver for this agent using default options.
func (ag *Agent) FileSystem() cloudfs.Driver {
	return ag.FileSystemWithOption(FileSystemOption{})
}

// FileSystemWithOption returns a cloudfs.Driver configured with the given options.
func (ag *Agent) FileSystemWithOption(opt FileSystemOption) cloudfs.Driver {
	if ag == nil || ag.Agent == nil {
		return nil
	}
	fs := NewFileSystemWithOption(ag.Agent, opt)
	if fs.limiter != nil {
		return cloudfs.NewRateLimitedDriver(fs, fs.limiter)
	}
	return fs
}

func (fs *FileSystem) Provider() string {
	return "115"
}

func (fs *FileSystem) Root(_ context.Context) (cloudfs.Entry, error) {
	return cloudfs.Entry{
		ID:   fs.rootID,
		Name: "/",
		Type: cloudfs.EntryTypeDirectory,
	}, nil
}

func (fs *FileSystem) Stat(ctx context.Context, entryID string) (cloudfs.Entry, error) {
	if entryID == "" || entryID == fs.rootID {
		return fs.Root(ctx)
	}
	file := &elevengo.File{}
	if err := fs.agent.FileGet(entryID, file); err != nil {
		return cloudfs.Entry{}, mapError(err)
	}
	return entryFromFile(file), nil
}

func (fs *FileSystem) List(ctx context.Context, dirID string) ([]cloudfs.Entry, error) {
	if dirID == "" {
		dirID = fs.rootID
	}
	it, err := fs.agent.FileIterate(dirID)
	if err != nil {
		return nil, mapError(err)
	}
	// pageSize is reserved; elevengo does not expose per-call limit control.
	items := make([]cloudfs.Entry, 0, it.Count())
	for _, file := range it.Items() {
		items = append(items, entryFromFile(file))
	}
	return items, nil
}

func (fs *FileSystem) Lookup(ctx context.Context, parentID, name string) (cloudfs.Entry, error) {
	items, err := fs.List(ctx, parentID)
	if err != nil {
		return cloudfs.Entry{}, err
	}
	var matched *cloudfs.Entry
	for _, item := range items {
		if item.Name != name {
			continue
		}
		if matched != nil {
			return cloudfs.Entry{}, cloudfs.ErrAmbiguousPath
		}
		copy := item
		matched = &copy
	}
	if matched == nil {
		return cloudfs.Entry{}, cloudfs.ErrNotFound
	}
	return *matched, nil
}

func (fs *FileSystem) Mkdir(ctx context.Context, parentID, name string) (cloudfs.Entry, error) {
	if parentID == "" {
		parentID = fs.rootID
	}
	dirID, err := fs.agent.DirMake(parentID, name)
	if err != nil {
		return cloudfs.Entry{}, mapError(err)
	}
	return fs.Stat(ctx, dirID)
}

func (fs *FileSystem) Rename(ctx context.Context, entryID, newName string) (cloudfs.Entry, error) {
	if err := fs.agent.FileRename(entryID, newName); err != nil {
		return cloudfs.Entry{}, mapError(err)
	}
	return fs.Stat(ctx, entryID)
}

func (fs *FileSystem) Move(ctx context.Context, targetDirID, entryID string) (cloudfs.Entry, error) {
	if targetDirID == "" {
		targetDirID = fs.rootID
	}
	if err := fs.agent.FileMove(targetDirID, []string{entryID}); err != nil {
		return cloudfs.Entry{}, mapError(err)
	}
	return fs.Stat(ctx, entryID)
}

func (fs *FileSystem) Copy(_ context.Context, targetDirID, entryID string) error {
	if targetDirID == "" {
		targetDirID = fs.rootID
	}
	if err := fs.agent.FileCopy(targetDirID, []string{entryID}); err != nil {
		return mapError(err)
	}
	return nil
}

func (fs *FileSystem) Delete(_ context.Context, entryID string) error {
	if err := fs.agent.FileDelete([]string{entryID}); err != nil {
		return mapError(err)
	}
	return nil
}

func (fs *FileSystem) Search(_ context.Context, dirID, keyword string, opts cloudfs.SearchOptions) ([]cloudfs.Entry, error) {
	var listOpt *option.FileListOptions
	switch {
	case opts.ExtName != "":
		listOpt = option.FileList().OnlyExtension(strings.TrimPrefix(opts.ExtName, "."))
	case opts.FileType > 0:
		listOpt = option.FileList()
		switch opts.FileType {
		case 1:
			listOpt.OnlyDocument()
		case 2:
			listOpt.OnlyImage()
		case 3:
			listOpt.OnlyAudio()
		case 4:
			listOpt.OnlyVideo()
		case 5:
			listOpt.OnlyArchive()
		case 6:
			listOpt.OnlySoftware()
		default:
			listOpt.ShowAll()
		}
	}

	var (
		it  elevengo.Iterator[elevengo.File]
		err error
	)
	if listOpt != nil {
		it, err = fs.agent.FileSearch(dirID, keyword, listOpt)
	} else {
		it, err = fs.agent.FileSearch(dirID, keyword)
	}
	if err != nil {
		return nil, mapError(err)
	}

	items := make([]cloudfs.Entry, 0, it.Count())
	for _, file := range it.Items() {
		entry := entryFromFile(file)
		if entry.IsDir() && !opts.IncludeDirectories {
			continue
		}
		items = append(items, entry)
	}
	return items, nil
}

// mapError normalises 115 SDK errors to cloudfs sentinel errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, llErrors.ErrNotExist):
		return cloudfs.ErrNotFound
	case errors.Is(err, llErrors.ErrExist):
		return cloudfs.ErrAlreadyExists
	case errors.Is(err, llErrors.ErrInvalidOperation):
		return cloudfs.ErrNotDirectory
	case errors.Is(err, llErrors.ErrInvalidParameters):
		return cloudfs.ErrInvalidName
	}
	var apiErr *llErrors.ApiError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 20004, 21003:
			return cloudfs.ErrAlreadyExists
		case 50003:
			return cloudfs.ErrNotFound
		case 20022:
			return cloudfs.ErrNotDirectory
		}
	}
	return cloudfs.ErrProviderFailure
}

func entryFromFile(file *elevengo.File) cloudfs.Entry {
	entryType := cloudfs.EntryTypeFile
	if file.IsDirectory {
		entryType = cloudfs.EntryTypeDirectory
	}
	return cloudfs.Entry{
		ID:       file.FileId,
		ParentID: file.ParentId,
		Name:     file.Name,
		Type:     entryType,
		Size:     file.Size,
		PickCode: file.PickCode,
	}
}

// NewOperationLimiter builds a CooldownLimiter from millisecond bounds.
// Returns nil if both values are zero (no limiting).
func NewOperationLimiter(minMs, maxMs int) cloudfs.Limiter {
	if minMs <= 0 && maxMs <= 0 {
		return nil
	}
	min := time.Duration(minMs) * time.Millisecond
	max := time.Duration(maxMs) * time.Millisecond
	return cloudfs.NewCooldownLimiter(min, max)
}
