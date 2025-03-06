package modules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Based on https://github.com/twpayne/find-duplicates, Tom knows his stuff so 1024 must be a good number:
const channelBufferCapacity = 1024

var (
	// ErrModuleSearch is generated when there are issues with finding modules
	// either an invalid path or
	ErrModuleSearch = fmt.Errorf("modules: Unable to search path for modules")
)

// findVersionsYamlFilesInPath concurrently looks for versions.yaml
// files starting from the given path down each folder recursively.
func findVersionsYamlFilesInPath(basePath string) ([]string, error) {
	if !filepath.IsAbs(basePath) {
		return nil, fmt.Errorf("%w basePath is not absolute: %s", ErrModuleSearch, basePath)
	}

	errCh := make(chan error, channelBufferCapacity)
	possibleVersionsFilesCh := make(chan string, channelBufferCapacity)

	walkDirFunc := func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
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
		switch err := walkDirFunc(path, dirEntry, nil); {
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
