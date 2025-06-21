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
	
	// Object pools for performance
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

	// Optimal worker count for I/O-bound operations: CPU cores * 3
	numWorkers := runtime.NumCPU() * 3
	if numWorkers > 16 {
		numWorkers = 16 // Cap at 16 to avoid scheduling overhead
	}
	if numWorkers < 4 {
		numWorkers = 4
	}

	scanner := &Scanner{
		workingDir: wd,
		targets:    make([]CleanupTarget, 0, 64), // Pre-allocate with reasonable capacity
		numWorkers: numWorkers,
	}
	
	// Initialize object pool
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

// High-performance directory size calculation using WalkDir
func (s *Scanner) calculateDirSize(dirPath string) int64 {
	var size int64
	
	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		// Use DirEntry.Type() to avoid expensive stat calls
		if !d.Type().IsDir() {
			if info, err := d.Info(); err == nil {
				size += info.Size()
			}
		}
		
		return nil
	})
	
	return size
}

func (s *Scanner) Scan() error {
	startTime := time.Now()
	
	s.targetsMutex.Lock()
	s.targets = s.targets[:0] // Reuse existing slice
	s.targetsMutex.Unlock()
	
	err := s.parallelScan(s.workingDir)
	s.scanDuration = time.Since(startTime)
	
	return err
}

func (s *Scanner) parallelScan(rootDir string) error {
	// Channel buffering: 2x worker count for optimal throughput
	bufferSize := s.numWorkers * 2
	workQueue := make(chan workItem, bufferSize)
	resultQueue := make(chan scanResult, bufferSize)
	
	var wg sync.WaitGroup

	// Start workers
	wg.Add(s.numWorkers)
	for i := 0; i < s.numWorkers; i++ {
		go s.worker(workQueue, resultQueue, &wg)
	}

	// Close result queue when all workers are done
	go func() {
		wg.Wait()
		close(resultQueue)
	}()

	// Start directory walker
	go func() {
		defer close(workQueue)
		s.walkDirectory(rootDir, workQueue)
	}()

	// Collect results
	for result := range resultQueue {
		if result.err != nil {
			continue
		}
		
		if result.target != nil && result.target.Path != "" {
			s.targetsMutex.Lock()
			s.targets = append(s.targets, *result.target)
			s.targetsMutex.Unlock()
			
			// Return target to pool
			s.targetPool.Put(result.target)
		}
	}

	return nil
}

// High-performance directory walking using WalkDir
func (s *Scanner) walkDirectory(dir string, workQueue chan<- workItem) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		
		// Use DirEntry.Type() to avoid expensive stat calls
		if d.Type().IsDir() {
			name := d.Name()
			
			// Early termination: check if this is a cleanup target
			if s.isCleanupTarget(name) {
				// Send to worker for processing
				select {
				case workQueue <- workItem{path: path, entry: d}:
				default:
					// Queue full, skip this item (graceful degradation)
				}
				
				// Skip scanning inside cleanup targets
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
		
		if s.isCleanupTarget(name) {
			// Get target from pool
			target := s.targetPool.Get().(*CleanupTarget)
			
			// Calculate size efficiently
			size := s.calculateDirSize(item.path)
			
			// Populate target
			target.Path = item.path
			target.Name = name
			target.Size = size
			target.Type = s.getTargetType(name)
			target.Selected = false
			
			resultQueue <- scanResult{target: target, err: nil}
		}
	}
}

// Optimized cleanup target detection using map lookup
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
	
	// Return a copy to prevent race conditions
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
