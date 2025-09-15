package scanner_compressfile

import (
	"errors"
	"io"
	"io/fs"
	"sync"

	"github.com/gen2brain/go-unarr"
)

type wrappedFS struct {
	archive       *unarr.Archive
	mut           *sync.Mutex
	info          fs.FileInfo
	subpath       string
	isClosed      bool
	archiveOffset int64

	resetSeek bool
	pos       int64
	newPos    int64
}

// Seek implements io.ReadSeeker.
func (w *wrappedFS) Seek(offset int64, whence int) (int64, error) {

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = w.pos + offset
	case io.SeekEnd:
		newPos = w.info.Size() - offset
	default:
		return 0, errors.New("invalid whence")
	}

	if newPos < 0 {
		return 0, errors.New("negative position")
	}

	if newPos != w.pos {
		w.newPos = newPos
		w.resetSeek = true
	}

	return newPos, nil
}

// Stat implements fs.File.
func (w *wrappedFS) Stat() (fs.FileInfo, error) {
	return w.info, nil
}

// Open implements fs.FS.
func (w *wrappedFS) Open(name string) (fs.File, error) {
	w.mut.Lock()
	defer w.mut.Unlock()

	err := w.archive.EntryFor(w.subpath)
	if err != nil {
		return nil, err
	}

	w.archiveOffset = w.archive.Offset()
	w.info = &ArchiveFile{
		isDir:    false,
		name:     w.archive.Name(),
		fileMode: 0,
		size:     int64(w.archive.Size()),
		modTime:  w.archive.ModTime(),
		offset:   w.archive.Offset(),
	}
	return w, nil
}

func (w *wrappedFS) Read(p []byte) (n int, err error) {
	if w.isClosed {
		return 0, io.ErrClosedPipe
	}

	w.mut.Lock()
	defer w.mut.Unlock()

	if w.archiveOffset != w.archive.Offset() || (w.resetSeek && w.newPos < w.pos) {
		err := w.archive.EntryAt(w.archiveOffset)
		if err != nil {
			return 0, err
		}

		if !w.resetSeek {
			w.newPos = w.pos
		}

		w.pos = 0
		w.resetSeek = true
	}

	if w.resetSeek {
		// move forward to new pos by using read only
		b := make([]byte, 32*1024)

		for w.pos < w.newPos {
			i := len(b)
			if w.pos+int64(len(b)) > w.newPos {
				i = int(w.newPos - w.pos)
			}

			n, err := w.archive.Read(b[:i])
			if err != nil {
				return 0, err
			}
			w.pos += int64(n)
		}

		w.resetSeek = false
	}

	i := len(p)
	if w.pos+int64(len(p)) > w.info.Size() {
		i = int(w.info.Size() - w.pos)
	}

	if i == 0 {
		return 0, io.EOF
	}

	n, err = w.archive.Read(p[:i])
	if err != nil {
		return 0, err
	}
	w.pos += int64(n)
	return n, err
}

func (w *wrappedFS) Close() error {
	w.isClosed = true
	return nil
}

var _ fs.FS = (*wrappedFS)(nil)
var _ fs.File = (*wrappedFS)(nil)
var _ io.ReadSeeker = (*wrappedFS)(nil)

func NewWrappedFS(archive *unarr.Archive, mut *sync.Mutex, subpath string) *wrappedFS {
	return &wrappedFS{
		archive: archive,
		mut:     mut,
		subpath: subpath,
	}
}
