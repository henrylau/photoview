package scanner_compressfile

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/go-unarr"
	"github.com/photoview/photoview/api/scanner/media_type"
)

type CompressFileFormat string

const (
	Zip    CompressFileFormat = "ZIP"
	SevenZ CompressFileFormat = "7Z"
	RAR    CompressFileFormat = "RAR"
)

var compressFile = map[string]CompressFileFormat{
	".zip": Zip,
	".7z":  SevenZ,
	".rar": RAR,
}

var compressFilePattern = regexp.MustCompile(`^(.*\.(zip|7z|rar))(.*)$`)

func IsArchiveFile(filePath string) bool {
	_, ok := compressFile[filepath.Ext(filePath)]
	return ok
}

func GetFileType(fileName string) (media_type.MediaType, error) {
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".jpeg":
		return media_type.TypeJPEG, nil
	case ".jpg":
		return media_type.TypeJPEG, nil
	case ".png":
		return media_type.TypePNG, nil
	case ".gif":
		return media_type.TypeGIF, nil
	case ".avif":
		return media_type.TypeAVIF, nil
	default:
		return media_type.TypeUnknown, fmt.Errorf("unknown media type (%s)", fileName)
	}
}

func IsArchiveFilePath(filePath string) bool {
	return compressFilePattern.MatchString(filePath)
}

func IsArchiveFileContainsPhotos(archivePath string) bool {
	archive, err := unarr.NewArchive(archivePath)
	if err != nil {
		log.Printf("Error opening archive: %s", err)
		return false
	}
	defer archive.Close()

	files, err := archive.List()
	if err != nil {
		log.Printf("Error listing archive contents: %s", err)
		return false
	}

	for _, file := range files {
		mediaType, err := GetFileType(file)
		if err == nil && mediaType != media_type.TypeUnknown {
			return true
		}
	}

	return false
}

func extractArchiveSubpath(archivePath string) (string, string, error) {
	data := compressFilePattern.FindStringSubmatch(archivePath)
	if data == nil || len(data) < 4 {
		return "", "", fmt.Errorf("invalid archive path (%s)", archivePath)
	}

	subpath := strings.TrimPrefix(data[3], "/")
	path := data[1]

	return path, subpath, nil
}

type archiveFileLoader struct {
	mut     sync.Mutex
	path    string
	archive *unarr.Archive
}

type loader struct {
	cache          map[string]map[string]ArchiveEntry
	loaderCache    map[string]*archiveFileLoader
	binaryCache    map[string][]byte
	binaryCacheMut sync.Mutex
}

func (l *loader) insertCache(archivePath string, data []byte) {
	l.binaryCacheMut.Lock()
	defer l.binaryCacheMut.Unlock()

	l.binaryCache[archivePath] = data

	// delay clear cache
	go func() {
		time.Sleep(15 * time.Second)
		l.binaryCacheMut.Lock()
		delete(l.binaryCache, archivePath)
		l.binaryCacheMut.Unlock()
	}()
}
func (l *loader) getCache(archivePath string) ([]byte, bool) {
	l.binaryCacheMut.Lock()
	defer l.binaryCacheMut.Unlock()

	data, found := l.binaryCache[archivePath]
	return data, found
}

func (l *loader) ListArchiveFiles(archivePath string) ([]ArchiveEntry, error) {
	if files, found := l.cache[archivePath]; found {
		result := make([]ArchiveEntry, len(files))

		i := 0
		for _, file := range files {
			result[i] = file
			i++
		}
		return result, nil
	}

	archive, err := unarr.NewArchive(archivePath)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	files := map[string]ArchiveEntry{}
	results := []ArchiveEntry{}
	for {
		err := archive.Entry()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		// Skip non-media files
		if _, err := GetFileType(archive.Name()); err != nil {
			continue
		}

		file := ArchiveFile{
			isDir:    false,
			name:     archive.Name(),
			fileMode: 0,
			size:     int64(archive.Size()),
			modTime:  archive.ModTime(),
			offset:   archive.Offset(),
		}

		files[archive.Name()] = &file
		results = append(results, &file)
	}
	l.cache[archivePath] = files

	// delay clear cache
	go func() {
		time.Sleep(10 * time.Minute)
		delete(l.cache, archivePath)
	}()

	return results, nil
}

func (l *loader) ListArchiveFilesBySubpath(archivePath string) ([]ArchiveEntry, error) {
	path, subpath, err := extractArchiveSubpath(archivePath)

	if err != nil {
		return nil, err
	}

	files, err := l.ListArchiveFiles(path)
	if err != nil {
		return nil, err
	}
	results := []ArchiveEntry{}
	for _, file := range files {
		if filepath.Dir(file.ArchivePath()) == subpath {
			results = append(results, file)
		}
	}
	return results, nil
}

func (l *loader) getLoader(path string) (*archiveFileLoader, error) {
	loader, ok := l.loaderCache[path]
	if !ok {
		archive, err := unarr.NewArchive(path)
		if err != nil {
			return nil, err
		}

		loader = &archiveFileLoader{
			archive: archive,
			path:    path,
			mut:     sync.Mutex{},
		}
		l.loaderCache[path] = loader

		// delay clear cache
		go func() {
			time.Sleep(10 * time.Minute)
			loader.mut.Lock()
			loader.archive.Close()
			loader.mut.Unlock()

			delete(l.loaderCache, path)
		}()
	}

	return loader, nil
}

func (l *loader) LoadFile(archivePath string) ([]byte, error) {
	path, subpath, err := extractArchiveSubpath(archivePath)
	if err != nil {
		return nil, err
	}

	if data, found := l.getCache(archivePath); found {
		return data, nil
	}

	loader, err := l.getLoader(path)
	if err != nil {
		return nil, err
	}

	loader.mut.Lock()
	defer loader.mut.Unlock()

	if l.cache[path] != nil && l.cache[path][subpath] != nil {
		entry := l.cache[path][subpath]
		err := loader.archive.EntryAt(entry.Offset())
		if err != nil {
			return nil, err
		}
	} else {
		err := loader.archive.EntryFor(subpath)
		if err != nil {
			return nil, err
		}
	}
	data, err := loader.archive.ReadAll()
	if err != nil {
		return nil, err
	}

	l.insertCache(archivePath, data)

	return data, nil
}

var Loader = loader{
	cache:          make(map[string]map[string]ArchiveEntry),
	binaryCache:    make(map[string][]byte),
	binaryCacheMut: sync.Mutex{},
	loaderCache:    make(map[string]*archiveFileLoader),
}
