package cloudfs

import "context"

type EntryType string

const (
	EntryTypeFile      EntryType = "file"
	EntryTypeDirectory EntryType = "directory"
)

type Entry struct {
	ID       string
	ParentID string
	Name     string
	Type     EntryType
	Size     int64
	PickCode string
}

func (e Entry) IsDir() bool {
	return e.Type == EntryTypeDirectory
}

type SearchOptions struct {
	FileType           int
	ExtName            string
	IncludeDirectories bool
}

type Reader interface {
	Provider() string
	Root(ctx context.Context) (Entry, error)
	Stat(ctx context.Context, entryID string) (Entry, error)
	List(ctx context.Context, dirID string) ([]Entry, error)
	Lookup(ctx context.Context, parentID, name string) (Entry, error)
}

type Mkdirer interface {
	Mkdir(ctx context.Context, parentID, name string) (Entry, error)
}

type Renamer interface {
	Rename(ctx context.Context, entryID, newName string) (Entry, error)
}

type Mover interface {
	Move(ctx context.Context, targetDirID, entryID string) (Entry, error)
}

type Copier interface {
	Copy(ctx context.Context, targetDirID, entryID string) error
}

type Remover interface {
	Delete(ctx context.Context, entryID string) error
}

type Searcher interface {
	Search(ctx context.Context, dirID, keyword string, opts SearchOptions) ([]Entry, error)
}

// Driver is the aggregate facade combining all capability interfaces.
type Driver interface {
	Reader
	Mkdirer
	Renamer
	Mover
	Copier
	Remover
}
