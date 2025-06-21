package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkScan(b *testing.B) {
	// Create a temporary directory with test structure
	tempDir, err := os.MkdirTemp("", "scanner_bench_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a complex directory structure with multiple targets
	testStructure := []string{
		"project1/node_modules",
		"project1/src",
		"project1/.next",
		"project2/node_modules",
		"project2/dist", 
		"project3/.nuxt",
		"project3/coverage",
		"project4/.cache",
		"project4/.vite",
		"deep/nested/project/node_modules",
	}

	for _, path := range testStructure {
		fullPath := filepath.Join(tempDir, path)
		err := os.MkdirAll(fullPath, 0755)
		if err != nil {
			b.Fatalf("Failed to create directory %s: %v", path, err)
		}

		// Add some files to make size calculation realistic
		testFile := filepath.Join(fullPath, "test.txt")
		err = os.WriteFile(testFile, []byte("test content for benchmarking"), 0644)
		if err != nil {
			b.Fatalf("Failed to create test file in %s: %v", path, err)
		}
	}

	// Change to temp directory
	originalWd, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	if err != nil {
		b.Fatalf("Failed to change to temp dir: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		scanner, err := New()
		if err != nil {
			b.Fatalf("Failed to create scanner: %v", err)
		}

		err = scanner.Scan()
		if err != nil {
			b.Fatalf("Failed to scan: %v", err)
		}

		targets := scanner.GetTargets()
		if len(targets) < 6 { // We expect at least 6 cleanup targets
			b.Fatalf("Expected at least 6 targets, got %d", len(targets))
		}
	}
}

func BenchmarkCalculateDirSize(b *testing.B) {
	// Create temp directory with files
	tempDir, err := os.MkdirTemp("", "size_bench_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory to make it the working directory
	originalWd, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	if err != nil {
		b.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Create multiple files and subdirectories
	for i := 0; i < 100; i++ {
		subDir := filepath.Join(tempDir, "sub", "dir", "level", "deep")
		err := os.MkdirAll(subDir, 0755)
		if err != nil {
			b.Fatalf("Failed to create subdirectory: %v", err)
		}

		testFile := filepath.Join(subDir, "file.txt")
		content := "This is test content for benchmarking directory size calculation"
		err = os.WriteFile(testFile, []byte(content), 0644)
		if err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	scanner, err := New()
	if err != nil {
		b.Fatalf("Failed to create scanner: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		size := scanner.calculateDirSize(tempDir)
		if size == 0 {
			b.Fatal("Expected non-zero directory size")
		}
	}
}