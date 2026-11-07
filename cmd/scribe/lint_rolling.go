package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (l *LintCmd) runRollingOnly() error {
	if !l.Fix {
		return errors.New("--rolling-only requires --fix")
	}
	if !l.DryRun {
		return errors.New("rolling-only apply is not available yet; use --fix --rolling-only --dry-run")
	}
	if l.Contradictions || l.Duplicates || l.Resolve || l.Identities || l.ApplyIdentities {
		return errors.New("--rolling-only cannot be combined with another lint operation")
	}
	root, err := kbDir()
	if err != nil {
		return err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return err
	}
	cfg := loadConfig(root)
	if err := cfg.requireParseable(); err != nil {
		return err
	}
	lockPath := lockPathFor(cfg.LockDir, "sync", root)
	lf, ok, err := acquireOperationLock(lockPath, "rolling-only preview")
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %s holds the selected KB sync lock", errLockBusy, lockOwnerDetails(lockPath))
	}
	defer releaseLock(lf)
	return l.previewRollingFiles(root)
}

func (l *LintCmd) previewRollingFiles(root string) error {
	var files []string
	var err error
	if l.Changed || len(l.Files) > 0 {
		files, err = l.targetFiles(root)
	} else {
		files = rollingPreviewFiles(root)
	}
	if err != nil {
		return err
	}
	sort.Strings(files)
	scanned, repairs, unchanged, unresolved := 0, 0, 0, 0
	seen := make(map[string]bool)
	for _, file := range files {
		path, rel, err := rollingPreviewPath(root, file)
		if seen[path] {
			continue
		}
		seen[path] = true
		var data []byte
		if err == nil {
			data, err = readRegularFile(path)
		}
		if err == nil && (!strings.HasSuffix(rel, ".md") || !isSessionRollingFile(root, rel, data)) {
			continue
		}
		scanned++
		var proposal []byte
		if err == nil {
			proposal, err = repairSessionRolling(root, rel, data)
		}
		switch {
		case err != nil:
			unresolved++
			fmt.Printf("unresolved %s: %v\n", filepath.ToSlash(rel), err)
		case bytes.Equal(data, proposal):
			unchanged++
			fmt.Printf("unchanged %s\n", filepath.ToSlash(rel))
		default:
			repairs++
			fmt.Printf("would repair %s\n", filepath.ToSlash(rel))
		}
	}
	fmt.Printf("scanned %d, would repair %d, unchanged %d, unresolved %d\n", scanned, repairs, unchanged, unresolved)
	if unresolved != 0 {
		return fmt.Errorf("rolling preview: %d unresolved file(s)", unresolved)
	}
	return nil
}

func rollingPreviewPath(root, file string) (string, string, error) {
	path, err := filepath.Abs(file)
	if err != nil {
		return file, file, err
	}
	rel, err := filepath.Rel(root, path)
	if err == nil && (rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		if canonicalRoot, rootErr := filepath.EvalSymlinks(root); rootErr == nil {
			rel, err = filepath.Rel(canonicalRoot, path)
		}
	}
	if err != nil {
		return path, file, err
	}
	guarded, err := rollingRestorePath(root, rel)
	if err != nil {
		return path, rel, err
	}
	return guarded, rel, nil
}

func rollingPreviewFiles(root string) []string {
	var files []string
	for _, dir := range wikiDirs {
		base := filepath.Join(root, dir)
		info, err := os.Lstat(base)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			files = append(files, base)
			continue
		}
		_ = filepath.WalkDir(base, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				files = append(files, path)
				return nil
			}
			if entry.IsDir() {
				return nil
			}
			if strings.HasSuffix(entry.Name(), ".md") {
				files = append(files, path)
			} else if entry.Type()&os.ModeSymlink != 0 {
				target, err := os.Stat(path)
				if err != nil || target.IsDir() {
					files = append(files, path)
				}
			}
			return nil
		})
	}
	return files
}
