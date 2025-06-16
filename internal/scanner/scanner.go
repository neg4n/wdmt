package scanner

import (
	"fmt"
	"os"
	"path/filepath"
)

type CleanupTarget struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Type     string `json:"type"`
	Selected bool   `json:"selected"`
}

type Scanner struct {
	workingDir       string
	targets          []CleanupTarget
	progressCallback func(float64)
	totalDirs        int
	processedDirs    int
}

var CommonCleanupDirs = map[string]string{
	"node_modules":  "Node.js/Bun.js dependencies",
	".next":         "Next.js build cache",
	"dist":          "Distribution/build files",
	".nuxt":         "Nuxt.js build cache",
	".output":       "Nuxt 3 output",
	".cache":        "Cache directory",
	"coverage":      "Test coverage reports",
	".nyc_output":   "NYC test coverage",
	"tmp":           "Temporary files",
	"temp":          "Temporary files",
	".parcel-cache": "Parcel bundler cache",
	".turbo":        "Turborepo cache",
	".webpack":      "Webpack cache",
	".rollup.cache": "Rollup cache",
	".vite":         "Vite cache",
	".swc":          "SWC cache",
	"lib-cov":       "Library coverage",
	".DS_Store":     "macOS metadata",
	"Thumbs.db":     "Windows metadata",
}

func New() (*Scanner, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	return &Scanner{
		workingDir: wd,
		targets:    make([]CleanupTarget, 0),
	}, nil
}

func (s *Scanner) ScanWithProgress(progressCallback func(float64)) error {
	s.targets = make([]CleanupTarget, 0)
	s.progressCallback = progressCallback
	s.processedDirs = 0

	s.totalDirs = s.countDirectories(s.workingDir)

	if s.progressCallback != nil {
		s.progressCallback(0.0)
	}

	err := s.scanRecursiveWithProgress(s.workingDir)

	if s.progressCallback != nil {
		s.progressCallback(1.0)
	}

	return err
}

func (s *Scanner) countDirectories(dir string) int {
	count := 0
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

func (s *Scanner) scanRecursiveWithProgress(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		s.processedDirs++
		if s.progressCallback != nil && s.totalDirs > 0 {
			progress := float64(s.processedDirs) / float64(s.totalDirs)
			if progress > 1.0 {
				progress = 1.0
			}
			s.progressCallback(progress)
		}

		if s.isCleanupTarget(info.Name()) {
			size, err := s.calculateDirSize(path)
			if err != nil {
				size = 0
			}

			target := CleanupTarget{
				Path:     path,
				Name:     info.Name(),
				Size:     size,
				Type:     s.getTargetType(info.Name()),
				Selected: false,
			}

			s.targets = append(s.targets, target)

			return filepath.SkipDir
		}

		return nil
	})
}

func (s *Scanner) Scan() error {
	s.targets = make([]CleanupTarget, 0)
	return s.scanRecursive(s.workingDir)
}

func (s *Scanner) scanRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		if s.isCleanupTarget(info.Name()) {
			size, err := s.calculateDirSize(path)
			if err != nil {
				size = 0
			}

			target := CleanupTarget{
				Path:     path,
				Name:     info.Name(),
				Size:     size,
				Type:     s.getTargetType(info.Name()),
				Selected: false,
			}

			s.targets = append(s.targets, target)

			return filepath.SkipDir
		}

		return nil
	})
}

func (s *Scanner) isCleanupTarget(name string) bool {
	_, exists := CommonCleanupDirs[name]
	return exists
}

func (s *Scanner) getTargetType(name string) string {
	if desc, exists := CommonCleanupDirs[name]; exists {
		return desc
	}
	return "Unknown"
}

func (s *Scanner) calculateDirSize(dirPath string) (int64, error) {
	var size int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})

	return size, err
}

func (s *Scanner) GetTargets() []CleanupTarget {
	return s.targets
}

func (s *Scanner) GetWorkingDir() string {
	return s.workingDir
}
