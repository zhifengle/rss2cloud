package p115

import (
	"errors"
	"testing"

	"github.com/deadblue/elevengo"
	llErrors "github.com/deadblue/elevengo/lowlevel/errors"
	"github.com/zhifengle/rss2cloud/cloudfs"
)

// --- entryFromFile ---

func TestEntryFromFile_File(t *testing.T) {
	f := &elevengo.File{
		FileId:      "123",
		ParentId:    "0",
		Name:        "test.mkv",
		IsDirectory: false,
		Size:        1024,
		PickCode:    "abc",
	}
	e := entryFromFile(f)
	if e.ID != "123" || e.ParentID != "0" || e.Name != "test.mkv" {
		t.Fatalf("unexpected entry: %+v", e)
	}
	if e.Type != cloudfs.EntryTypeFile {
		t.Fatalf("expected file type, got %v", e.Type)
	}
	if e.Size != 1024 || e.PickCode != "abc" {
		t.Fatalf("unexpected size/pickcode: %+v", e)
	}
}

func TestEntryFromFile_Directory(t *testing.T) {
	f := &elevengo.File{
		FileId:      "5",
		ParentId:    "0",
		Name:        "anime",
		IsDirectory: true,
	}
	e := entryFromFile(f)
	if e.Type != cloudfs.EntryTypeDirectory {
		t.Fatalf("expected directory type, got %v", e.Type)
	}
}

// --- mapError ---

func TestMapError_Nil(t *testing.T) {
	if mapError(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestMapError_ErrNotExist(t *testing.T) {
	err := mapError(llErrors.ErrNotExist)
	if !errors.Is(err, cloudfs.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMapError_ErrExist(t *testing.T) {
	err := mapError(llErrors.ErrExist)
	if !errors.Is(err, cloudfs.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestMapError_ErrInvalidOperation(t *testing.T) {
	err := mapError(llErrors.ErrInvalidOperation)
	if !errors.Is(err, cloudfs.ErrNotDirectory) {
		t.Fatalf("expected ErrNotDirectory, got %v", err)
	}
}

func TestMapError_ErrInvalidParameters(t *testing.T) {
	err := mapError(llErrors.ErrInvalidParameters)
	if !errors.Is(err, cloudfs.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}

func TestMapError_ApiErrorExist(t *testing.T) {
	err := mapError(&llErrors.ApiError{Code: 20004})
	if !errors.Is(err, cloudfs.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists for code 20004, got %v", err)
	}
}

func TestMapError_ApiErrorNotExist(t *testing.T) {
	err := mapError(&llErrors.ApiError{Code: 50003})
	if !errors.Is(err, cloudfs.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for code 50003, got %v", err)
	}
}

func TestMapError_Unknown(t *testing.T) {
	err := mapError(errors.New("some unexpected error"))
	if !errors.Is(err, cloudfs.ErrProviderFailure) {
		t.Fatalf("expected ErrProviderFailure, got %v", err)
	}
}

// --- Root ---

func TestFileSystem_Root(t *testing.T) {
	fs := NewFileSystem(&elevengo.Agent{})
	e, err := fs.Root(nil)
	if err != nil {
		t.Fatalf("Root() error: %v", err)
	}
	if e.ID != defaultRootID || e.Type != cloudfs.EntryTypeDirectory || e.Name != "/" {
		t.Fatalf("unexpected root entry: %+v", e)
	}
}
