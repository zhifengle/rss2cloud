package cloudfs

import "errors"

var (
	ErrNotFound       = errors.New("cloudfs: not found")
	ErrNotDirectory   = errors.New("cloudfs: not a directory")
	ErrAlreadyExists  = errors.New("cloudfs: already exists")
	ErrInvalidPath    = errors.New("cloudfs: invalid path")
	ErrInvalidName    = errors.New("cloudfs: invalid name")
	ErrAmbiguousPath  = errors.New("cloudfs: ambiguous path")
	ErrUnsupported    = errors.New("cloudfs: unsupported operation")
	ErrRateLimited    = errors.New("cloudfs: rate limited")
	ErrProviderFailure = errors.New("cloudfs: provider failure")
)
