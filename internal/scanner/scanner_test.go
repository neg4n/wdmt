package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	scanner, err := New()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if scanner == nil {
		t.Fatal("Expected scanner to be created")
	}

	if scanner.workingDir == "" {
		t.Error("Expected working directory to be set")
	}

	if len(scanner.targets) != 0 {
		t.Error("Expected empty targets slice")
	}
}

func TestIsCleanupTarget(t *testing.T) {
	scanner, err := New()
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	tests := []struct {
		name     string
		dirname  string
		expected bool
	}{
		{"Node modules", "node_modules", true},
		{"Next.js cache", ".next", true},
		{"Distribution directory", "dist", true},
		{"Nuxt cache", ".nuxt", true},
		{"Coverage reports", "coverage", true},
		{"Regular directory", "src", false},
		{"User directory", "documents", false},
		{"Empty string", "", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := scanner.isCleanupTarget(test.dirname)
			if result != test.expected {
				t.Errorf("Expected isCleanupTarget(%s) to be %v, got %v",
					test.dirname, test.expected, result)
			}
		})
	}
}

func TestGetTargetType(t *testing.T) {
	scanner, err := New()
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	tests := []struct {
		name     string
		dirname  string
		expected string
	}{
		{"Node modules", "node_modules", "Node.js/Bun.js dependencies"},
		{"Next.js cache", ".next", "Next.js build cache"},
		{"Distribution directory", "dist", "Distribution/build files"},
		{"Unknown directory", "unknown", "Unknown"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := scanner.getTargetType(test.dirname)
			if result != test.expected {
				t.Errorf("Expected getTargetType(%s) to be %s, got %s",
					test.dirname, test.expected, result)
			}
		})
	}
}

func TestScanRecursive(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_test_recursive_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	projectDir := filepath.Join(tempDir, "project")
	err = os.MkdirAll(projectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	subProjectDir := filepath.Join(projectDir, "subproject")
	err = os.MkdirAll(subProjectDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subproject directory: %v", err)
	}

	testTargets := []string{
		filepath.Join(projectDir, "node_modules"),
		filepath.Join(subProjectDir, ".next"),
		filepath.Join(tempDir, "dist"),
	}

	for _, target := range testTargets {
		err := os.MkdirAll(target, 0755)
		if err != nil {
			t.Fatalf("Failed to create target directory %s: %v", target, err)
		}

		testFile := filepath.Join(target, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file in %s: %v", target, err)
		}
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	scanner, err := New()
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	err = scanner.Scan()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	targets := scanner.GetTargets()

	if len(targets) < 3 {
		t.Errorf("Expected at least 3 targets, got %d", len(targets))
		for i, target := range targets {
			t.Logf("Target %d: %s (%s)", i, target.Name, target.Path)
		}
	}

	expectedNames := []string{"node_modules", ".next", "dist"}
	foundNames := make(map[string]bool)
	for _, target := range targets {
		foundNames[target.Name] = true
	}

	for _, expected := range expectedNames {
		if !foundNames[expected] {
			t.Errorf("Expected to find target %s", expected)
		}
	}
}

func TestScan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_test_scan_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testDirs := []string{"node_modules", "src", ".next"}
	for _, dir := range testDirs {
		dirPath := filepath.Join(tempDir, dir)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}

		testFile := filepath.Join(dirPath, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file in %s: %v", dir, err)
		}
	}

	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	scanner, err := New()
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	err = scanner.Scan()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	targets := scanner.GetTargets()
	if len(targets) != 2 { 
		t.Errorf("Expected 2 targets, got %d", len(targets))
	}
}

func TestCalculateDirSizeConcurrent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_test_size_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile1 := filepath.Join(tempDir, "file1.txt")
	content1 := "hello world" 
	err = os.WriteFile(testFile1, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	testFile2 := filepath.Join(tempDir, "file2.txt")
	content2 := "test" 
	err = os.WriteFile(testFile2, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	subDir := filepath.Join(tempDir, "subdir")
	err = os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	testFile3 := filepath.Join(subDir, "file3.txt")
	content3 := "sub" 
	err = os.WriteFile(testFile3, []byte(content3), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file in subdirectory: %v", err)
	}

	scanner, err := New()
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	size := scanner.calculateDirSizeConcurrent(tempDir)

	expectedSize := int64(len(content1) + len(content2) + len(content3)) 
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestGetters(t *testing.T) {
	scanner, err := New()
	if err != nil {
		t.Fatalf("Failed to create scanner: %v", err)
	}

	workingDir := scanner.GetWorkingDir()
	if workingDir == "" {
		t.Error("Expected working directory to be set")
	}

	targets := scanner.GetTargets()
	if len(targets) != 0 {
		t.Error("Expected empty targets slice")
	}

	scanner.targets = append(scanner.targets, CleanupTarget{
		Path: "/test/path",
		Name: "test",
		Size: 100,
		Type: "Test",
	})

	targets = scanner.GetTargets()
	if len(targets) != 1 {
		t.Error("Expected one target")
	}

	if targets[0].Name != "test" {
		t.Error("Expected target name to be 'test'")
	}
}

func TestCommonCleanupDirs(t *testing.T) {
	if len(CommonCleanupDirs) == 0 {
		t.Error("Expected CommonCleanupDirs to contain entries")
	}

	expectedEntries := map[string]string{
		"node_modules": "Node.js/Bun.js dependencies",
		".next":        "Next.js build cache",
		"dist":         "Distribution/build files",
	}

	for name, expectedDesc := range expectedEntries {
		if desc, exists := CommonCleanupDirs[name]; !exists {
			t.Errorf("Expected CommonCleanupDirs to contain %s", name)
		} else if desc != expectedDesc {
			t.Errorf("Expected description for %s to be %s, got %s", name, expectedDesc, desc)
		}
	}
}
