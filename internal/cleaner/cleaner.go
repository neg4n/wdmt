package cleaner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"wdmt/internal/scanner"
)

type Cleaner struct {
	workingDir    string
	workingDirDev uint64 
}

type SecurityError struct {
	Path   string
	Reason string
}

func (e *SecurityError) Error() string {
	return fmt.Sprintf("security violation for path %s: %s", e.Path, e.Reason)
}

func New(workingDir string) (*Cleaner, error) {
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve working directory: %w", err)
	}

	stat, err := os.Lstat(absWorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat working directory: %w", err)
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("working directory is not a directory: %s", absWorkingDir)
	}

	if stat.Mode()&os.ModeSymlink != 0 {
		return nil, &SecurityError{
			Path:   absWorkingDir,
			Reason: "working directory cannot be a symlink",
		}
	}

	var workingDirDev uint64
	if sysstat, ok := stat.Sys().(*syscall.Stat_t); ok {
		workingDirDev = uint64(sysstat.Dev)
	}

	return &Cleaner{
		workingDir:    absWorkingDir,
		workingDirDev: workingDirDev,
	}, nil
}


func (c *Cleaner) secureDeleteDirectory(path string) error {
	if err := c.validatePathSecurity(path); err != nil {
		return err
	}

	stat, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}
	if err != nil {
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	if stat.Mode()&os.ModeSymlink != 0 {
		return &SecurityError{
			Path:   path,
			Reason: "target is a symlink, refusing to delete",
		}
	}

	if !stat.IsDir() {
		return &SecurityError{
			Path:   path,
			Reason: "target is not a directory",
		}
	}

	return c.secureRemoveAll(path)
}

func (c *Cleaner) secureRemoveAll(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open directory %s: %w", path, err)
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", path, err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(path, entry.Name())

		if err := c.validatePathSecurity(entryPath); err != nil {
			continue
		}

		if entry.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(entryPath); err != nil {
				continue
			}
		} else if entry.IsDir() {
			if err := c.secureRemoveAll(entryPath); err != nil {
				continue
			}
		} else {
			if err := os.Remove(entryPath); err != nil {
				continue
			}
		}
	}

	return os.Remove(path)
}

func (c *Cleaner) validatePathSecurity(path string) error {
	if !utf8.ValidString(path) {
		return &SecurityError{
			Path:   path,
			Reason: "path contains invalid UTF-8 characters",
		}
	}

	if strings.Contains(path, "\x00") {
		return &SecurityError{
			Path:   path,
			Reason: "path contains null bytes",
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	cleanPath := filepath.Clean(absPath)
	if absPath != cleanPath {
		return &SecurityError{
			Path:   path,
			Reason: "path normalization mismatch (potential traversal attack)",
		}
	}

	if !strings.HasPrefix(absPath+string(filepath.Separator), c.workingDir+string(filepath.Separator)) {
		return &SecurityError{
			Path:   path,
			Reason: "path is outside working directory",
		}
	}

	if absPath == c.workingDir {
		return &SecurityError{
			Path:   path,
			Reason: "cannot delete working directory itself",
		}
	}

	rel, err := filepath.Rel(c.workingDir, absPath)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}

	if strings.HasPrefix(rel, "..") || strings.Contains(rel, string(filepath.Separator)+"..") {
		return &SecurityError{
			Path:   path,
			Reason: "path attempts to traverse outside working directory",
		}
	}

	if c.workingDirDev != 0 {
		if stat, err := os.Lstat(absPath); err == nil {
			if sysstat, ok := stat.Sys().(*syscall.Stat_t); ok {
				if uint64(sysstat.Dev) != c.workingDirDev {
					return &SecurityError{
						Path:   path,
						Reason: "path crosses filesystem boundary",
					}
				}
			}
		}
	}

	return c.validatePathComponents(absPath)
}

func (c *Cleaner) validatePathComponents(path string) error {
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == current || parent == c.workingDir {
			break
		}

		if stat, err := os.Lstat(parent); err == nil {
			if stat.Mode()&os.ModeSymlink != 0 {
				return &SecurityError{
					Path:   path,
					Reason: fmt.Sprintf("parent directory %s is a symlink", parent),
				}
			}
		}

		current = parent
	}

	return nil
}

func (c *Cleaner) ValidateTargets(targets []scanner.CleanupTarget) ([]scanner.CleanupTarget, error) {
	var validTargets []scanner.CleanupTarget

	for _, target := range targets {
		if err := c.validatePathSecurity(target.Path); err != nil {
			continue
		}

		stat, err := os.Lstat(target.Path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			continue
		}

		if stat.Mode()&os.ModeSymlink != 0 {
			continue
		}

		if !stat.IsDir() {
			continue
		}

		validTargets = append(validTargets, target)
	}

	return validTargets, nil
}

