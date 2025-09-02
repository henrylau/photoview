package scanner_compressfile

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type ArchiveFile struct {
	isDir    bool
	name     string
	fileMode fs.FileMode
	size     int64
	modTime  time.Time
	offset   int64
}

// ArchivePath implements ArchiveEntry.
func (a *ArchiveFile) ArchivePath() string {
	return a.name
}

// Offset implements ArchiveEntry.
func (a *ArchiveFile) Offset() int64 {
	return a.offset
}

type ArchiveEntry interface {
	os.DirEntry
	ArchivePath() string
	Offset() int64
}

// ModTime implements fs.FileInfo.
func (a *ArchiveFile) ModTime() time.Time {
	return a.modTime
}

// Mode implements fs.FileInfo.
func (a *ArchiveFile) Mode() fs.FileMode {
	return a.fileMode
}

// Size implements fs.FileInfo.
func (a *ArchiveFile) Size() int64 {
	return a.size
}

// Sys implements fs.FileInfo.
func (a *ArchiveFile) Sys() any {
	return struct{ offset int64 }{offset: a.offset}
}

// Info implements fs.DirEntry.
func (a *ArchiveFile) Info() (fs.FileInfo, error) {
	return a, nil
}

// IsDir implements fs.DirEntry.
func (a *ArchiveFile) IsDir() bool {
	return a.isDir
}

// Name implements fs.DirEntry.
func (a *ArchiveFile) Name() string {
	return filepath.Base(a.name)
}

// Type implements fs.DirEntry.
func (a *ArchiveFile) Type() fs.FileMode {
	return a.fileMode
}

var _ ArchiveEntry = (*ArchiveFile)(nil)
var _ fs.FileInfo = (*ArchiveFile)(nil)
