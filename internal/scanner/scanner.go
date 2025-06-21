package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
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
	numWorkers       int
	targetsMutex     sync.RWMutex
	progressMutex    sync.RWMutex
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

	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}
	if numWorkers < 2 {
		numWorkers = 2
	}

	return &Scanner{
		workingDir: wd,
		targets:    make([]CleanupTarget, 0),
		numWorkers: numWorkers,
	}, nil
}

func (s *Scanner) ScanWithProgress(progressCallback func(float64)) error {
	s.targetsMutex.Lock()
	s.targets = make([]CleanupTarget, 0)
	s.targetsMutex.Unlock()
	
	s.progressMutex.Lock()
	s.progressCallback = progressCallback
	s.processedDirs = 0
	s.progressMutex.Unlock()

	if s.progressCallback != nil {
		s.progressCallback(0.0)
	}

	err := s.parallelScanWithProgress(s.workingDir)

	if s.progressCallback != nil {
		s.progressCallback(1.0)
	}

	return err
}

type workItem struct {
	path string
	info os.FileInfo
}

type scanResult struct {
	target CleanupTarget
	err    error
}

func (s *Scanner) parallelScanWithProgress(rootDir string) error {
	workQueue := make(chan workItem, 1000)
	resultQueue := make(chan scanResult, 100)
	var wg sync.WaitGroup
	var totalDirs int64
	var processedDirs int64

	wg.Add(s.numWorkers)
	for i := 0; i < s.numWorkers; i++ {
		go s.worker(workQueue, resultQueue, &wg, &processedDirs, &totalDirs)
	}

	go func() {
		wg.Wait()
		close(resultQueue)
	}()

	go func() {
		defer close(workQueue)
		s.walkDirectory(rootDir, workQueue, &totalDirs)
	}()

	for result := range resultQueue {
		if result.err != nil {
			continue
		}
		
		if result.target.Path != "" {
			s.targetsMutex.Lock()
			s.targets = append(s.targets, result.target)
			s.targetsMutex.Unlock()
		}

		if s.progressCallback != nil {
			processed := atomic.LoadInt64(&processedDirs)
			total := atomic.LoadInt64(&totalDirs)
			if total > 0 {
				progress := float64(processed) / float64(total)
				if progress > 1.0 {
					progress = 1.0
				}
				s.progressCallback(progress)
			}
		}
	}

	return nil
}

func (s *Scanner) walkDirectory(dir string, workQueue chan<- workItem, totalDirs *int64) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if info.IsDir() {
			atomic.AddInt64(totalDirs, 1)
			select {
			case workQueue <- workItem{path: path, info: info}:
			default:
			}
			
			if s.isCleanupTarget(info.Name()) {
				return filepath.SkipDir
			}
		}
		
		return nil
	})
}

func (s *Scanner) worker(workQueue <-chan workItem, resultQueue chan<- scanResult, wg *sync.WaitGroup, processedDirs *int64, totalDirs *int64) {
	defer wg.Done()
	
	for item := range workQueue {
		atomic.AddInt64(processedDirs, 1)
		
		if s.isCleanupTarget(item.info.Name()) {
			size := s.calculateDirSizeConcurrent(item.path)
			
			target := CleanupTarget{
				Path:     item.path,
				Name:     item.info.Name(),
				Size:     size,
				Type:     s.getTargetType(item.info.Name()),
				Selected: false,
			}
			
			resultQueue <- scanResult{target: target, err: nil}
		} else {
			resultQueue <- scanResult{target: CleanupTarget{}, err: nil}
		}
	}
}

func (s *Scanner) calculateDirSizeConcurrent(dirPath string) int64 {
	var size int64
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	semaphore := make(chan struct{}, s.numWorkers)
	
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if !info.IsDir() {
			wg.Add(1)
			go func(fileSize int64) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				mu.Lock()
				size += fileSize
				mu.Unlock()
			}(info.Size())
		}
		
		return nil
	})
	
	wg.Wait()
	return size
}



func (s *Scanner) Scan() error {
	s.targetsMutex.Lock()
	s.targets = make([]CleanupTarget, 0)
	s.targetsMutex.Unlock()
	
	return s.parallelScan(s.workingDir)
}

func (s *Scanner) parallelScan(rootDir string) error {
	workQueue := make(chan workItem, 1000)
	resultQueue := make(chan scanResult, 100)
	var wg sync.WaitGroup

	wg.Add(s.numWorkers)
	for i := 0; i < s.numWorkers; i++ {
		go s.workerNoProgress(workQueue, resultQueue, &wg)
	}

	go func() {
		wg.Wait()
		close(resultQueue)
	}()

	go func() {
		defer close(workQueue)
		s.walkDirectoryNoProgress(rootDir, workQueue)
	}()

	for result := range resultQueue {
		if result.err != nil {
			continue
		}
		
		if result.target.Path != "" {
			s.targetsMutex.Lock()
			s.targets = append(s.targets, result.target)
			s.targetsMutex.Unlock()
		}
	}

	return nil
}

func (s *Scanner) walkDirectoryNoProgress(dir string, workQueue chan<- workItem) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if info.IsDir() {
			select {
			case workQueue <- workItem{path: path, info: info}:
			default:
			}
			
			if s.isCleanupTarget(info.Name()) {
				return filepath.SkipDir
			}
		}
		
		return nil
	})
}

func (s *Scanner) workerNoProgress(workQueue <-chan workItem, resultQueue chan<- scanResult, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for item := range workQueue {
		if s.isCleanupTarget(item.info.Name()) {
			size := s.calculateDirSizeConcurrent(item.path)
			
			target := CleanupTarget{
				Path:     item.path,
				Name:     item.info.Name(),
				Size:     size,
				Type:     s.getTargetType(item.info.Name()),
				Selected: false,
			}
			
			resultQueue <- scanResult{target: target, err: nil}
		} else {
			resultQueue <- scanResult{target: CleanupTarget{}, err: nil}
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
