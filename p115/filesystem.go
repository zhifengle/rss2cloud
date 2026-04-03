package p115

import (
	"context"
	"errors"

	"github.com/deadblue/elevengo"
	llErrors "github.com/deadblue/elevengo/lowlevel/errors"
	"github.com/zhifengle/rss2cloud/cloudfs"
)

const rootDirID = "0"

type FileSystem struct {
	agent *elevengo.Agent
}

func NewFileSystem(agent *elevengo.Agent) *FileSystem {
	return &FileSystem{agent: agent}
}

func (ag *Agent) FileSystem() cloudfs.Driver {
	if ag == nil || ag.Agent == nil {
		return nil
	}
	return NewFileSystem(ag.Agent)
}

func (fs *FileSystem) Provider() string {
	return "115"
}

func (fs *FileSystem) Root(_ context.Context) (cloudfs.Entry, error) {
	return cloudfs.Entry{
		ID:   rootDirID,
		Name: "/",
		Type: cloudfs.EntryTypeDirectory,
	}, nil
}

func (fs *FileSystem) Stat(ctx context.Context, entryID string) (cloudfs.Entry, error) {
	if entryID == "" || entryID == rootDirID {
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
		dirID = rootDirID
	}
	it, err := fs.agent.FileIterate(dirID)
	if err != nil {
		return nil, mapError(err)
	}
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
		parentID = rootDirID
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
		targetDirID = rootDirID
	}
	if err := fs.agent.FileMove(targetDirID, []string{entryID}); err != nil {
		return cloudfs.Entry{}, mapError(err)
	}
	return fs.Stat(ctx, entryID)
}

func (fs *FileSystem) Copy(_ context.Context, targetDirID, entryID string) error {
	if targetDirID == "" {
		targetDirID = rootDirID
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

// mapError normalises 115 SDK errors to cloudfs sentinel errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	// Match elevengo sentinel errors first.
	switch {
	case errors.Is(err, llErrors.ErrNotExist):
		return cloudfs.ErrNotFound
	case errors.Is(err, llErrors.ErrExist):
		return cloudfs.ErrAlreadyExists
	case errors.Is(err, llErrors.ErrInvalidOperation):
		// 115 returns this for operations on wrong target types (e.g. non-dir).
		return cloudfs.ErrNotDirectory
	case errors.Is(err, llErrors.ErrInvalidParameters):
		return cloudfs.ErrInvalidName
	}
	// Fall back to ApiError code inspection.
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
