package modules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Based on https://github.com/twpayne/find-duplicates, Tom knows his stuff so 1024 must be a good number:
const channelBufferCapacity = 1024

var (
	// ErrModuleSearch is generated when there are issues with finding modules
	// either an invalid path or
	ErrModuleSearch = errors.New("modules: Unable to search path for modules")
)

// findVersionsYamlFilesInPath concurrently looks for versions.yaml
// files starting from the given path down each folder recursively.
// It does not search ignoredDirs, which must be clean absolute paths.
func findVersionsYamlFilesInPath(basePath string, ignoredDirs []string) ([]string, error) {
	if !filepath.IsAbs(basePath) {
		return nil, fmt.Errorf("%w basePath is not absolute: %s", ErrModuleSearch, basePath)
	}
	if isInAnyDir(filepath.Clean(basePath), ignoredDirs) {
		return []string{}, nil
	}

	errCh := make(chan error, channelBufferCapacity)
	possibleVersionsFilesCh := make(chan string, channelBufferCapacity)

	walkDirFunc := func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			// The walk goes top-down from a path outside ignoredDirs, so it
			// reaches an ignored directory itself before anything inside it.
			if slices.Contains(ignoredDirs, path) {
				return fs.SkipDir
			}
			return nil
		}
		basename := filepath.Base(path)
		if basename == "versions.yaml" || basename == "versions.yml" {
			possibleVersionsFilesCh <- path
		}

		return nil
	}

	go func() {
		defer close(possibleVersionsFilesCh)
		defer close(errCh)
		concurrentWalkDir(basePath, walkDirFunc, errCh)
	}()

	possibleVersionsFiles := make([]string, 0)
	var err error
	for possiblePath := range possibleVersionsFilesCh {
		possibleVersionsFiles = append(possibleVersionsFiles, possiblePath)
	}
	for pathErr := range errCh {
		err = errors.Join(err, fmt.Errorf("%w %w", ErrModuleSearch, pathErr))
	}
	return possibleVersionsFiles, err
}

// isInAnyDir reports whether path is one of dirs or inside one of them.
func isInAnyDir(path string, dirs []string) bool {
	for _, dir := range dirs {
		if path == dir || strings.HasPrefix(path, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func concurrentWalkDir(root string, walkDirFunc fs.WalkDirFunc, errCh chan<- error) {
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		errCh <- walkDirFunc(root, nil, err)
		return
	}
	files := 0
	for _, dirEntry := range dirEntries {
		if dirEntry.Type().IsRegular() {
			files++
		}
	}
	var wg sync.WaitGroup
CONCURENT_WALK_DIR_FOR:
	for _, dirEntry := range dirEntries {
		path := filepath.Join(root, dirEntry.Name())
		switch err := walkDirFunc(path, dirEntry, nil); { //nolint:revive
		case errors.Is(err, fs.SkipAll):
			break CONCURENT_WALK_DIR_FOR
		case dirEntry.IsDir() && errors.Is(err, fs.SkipDir):
			// Skip directory.
		case err != nil:
			errCh <- err
			return
		case dirEntry.IsDir():
			wg.Add(1)
			go func() {
				defer wg.Done()
				concurrentWalkDir(path, walkDirFunc, errCh)
			}()
		}
	}
	wg.Wait()
}
