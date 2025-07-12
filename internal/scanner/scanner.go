package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

type CleanupTarget struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Type     string `json:"type"`
	Selected bool   `json:"selected"`
}

type Scanner struct {
	workingDir   string
	targets      []CleanupTarget
	numWorkers   int
	targetsMutex sync.RWMutex
	scanDuration time.Duration

	targetPool sync.Pool
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

	numWorkers := runtime.NumCPU() * 3
	if numWorkers > 16 {
		numWorkers = 16
	}
	if numWorkers < 4 {
		numWorkers = 4
	}

	scanner := &Scanner{
		workingDir: wd,
		targets:    make([]CleanupTarget, 0, 64),
		numWorkers: numWorkers,
	}

	scanner.targetPool.New = func() interface{} {
		return &CleanupTarget{}
	}

	return scanner, nil
}

type workItem struct {
	path  string
	entry fs.DirEntry
}

type scanResult struct {
	target *CleanupTarget
	err    error
}

func (s *Scanner) calculateDirSize(dirPath string) int64 {
	var size int64
	const blockSize = 4096

	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				fileSize := info.Size()

				if fileSize == 0 {

					size += blockSize
				} else {

					blocks := (fileSize + blockSize - 1) / blockSize
					size += blocks * blockSize
				}
			}
		}

		return nil
	})

	return size
}

func (s *Scanner) Scan() error {
	startTime := time.Now()

	s.targetsMutex.Lock()
	s.targets = s.targets[:0]
	s.targetsMutex.Unlock()

	err := s.parallelScan(s.workingDir)
	s.scanDuration = time.Since(startTime)

	return err
}

func (s *Scanner) parallelScan(rootDir string) error {

	bufferSize := s.numWorkers * 2
	workQueue := make(chan workItem, bufferSize)
	resultQueue := make(chan scanResult, bufferSize)

	var wg sync.WaitGroup

	wg.Add(s.numWorkers)
	for i := 0; i < s.numWorkers; i++ {
		go s.worker(workQueue, resultQueue, &wg)
	}

	go func() {
		wg.Wait()
		close(resultQueue)
	}()

	go func() {
		defer close(workQueue)
		s.walkDirectory(rootDir, workQueue)
	}()

	for result := range resultQueue {
		if result.err != nil {
			continue
		}

		if result.target != nil && result.target.Path != "" {
			s.targetsMutex.Lock()
			s.targets = append(s.targets, *result.target)
			s.targetsMutex.Unlock()

			s.targetPool.Put(result.target)
		}
	}

	return nil
}

func (s *Scanner) walkDirectory(dir string, workQueue chan<- workItem) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		if d.Type().IsDir() {
			name := d.Name()

			if s.isCleanupTarget(name) {

				select {
				case workQueue <- workItem{path: path, entry: d}:
				default:

				}

				return filepath.SkipDir
			}
		}

		return nil
	})
}

func (s *Scanner) worker(workQueue <-chan workItem, resultQueue chan<- scanResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for item := range workQueue {
		name := item.entry.Name()

		if item.entry.Type()&fs.ModeSymlink != 0 {
			continue
		}

		if s.isCleanupTarget(name) {

			target := s.targetPool.Get().(*CleanupTarget)

			size := s.calculateDirSize(item.path)

			target.Path = item.path
			target.Name = name
			target.Size = size
			target.Type = s.getTargetType(name)
			target.Selected = false

			resultQueue <- scanResult{target: target, err: nil}
		}
	}
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

func (s *Scanner) GetTargets() []CleanupTarget {
	s.targetsMutex.RLock()
	defer s.targetsMutex.RUnlock()

	targets := make([]CleanupTarget, len(s.targets))
	copy(targets, s.targets)
	return targets
}

func (s *Scanner) GetWorkingDir() string {
	return s.workingDir
}

func (s *Scanner) GetScanDuration() time.Duration {
	return s.scanDuration
}

func (s *Scanner) GetScanDurationString() string {
	duration := s.scanDuration
	if duration < time.Second {
		return fmt.Sprintf("%.1fs", duration.Seconds())
	}
	return fmt.Sprintf("%.1fs", duration.Seconds())
}

func (s *Scanner) CalculateDirectorySize(dirPath string) int64 {
	return s.calculateDirSize(dirPath)
}
