package cleaner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neg4n/wdmt/internal/scanner"
)

func createSafeTestEnv(t *testing.T) (string, func()) {
	baseTemp, err := os.MkdirTemp("", "wdmt_safe_test")
	if err != nil {
		t.Fatalf("Failed to create base temp dir: %v", err)
	}

	safeTestRoot := filepath.Join(baseTemp, "wdmt_isolated", "test_workspace")
	err = os.MkdirAll(safeTestRoot, 0755)
	if err != nil {
		os.RemoveAll(baseTemp)
		t.Fatalf("Failed to create safe test root: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(baseTemp)
	}

	return safeTestRoot, cleanup
}

func TestNew_SecurityValidation(t *testing.T) {
	safeTestRoot, cleanup := createSafeTestEnv(t)
	defer cleanup()

	cleaner, err := New(safeTestRoot)
	if err != nil {
		t.Errorf("Expected no error for valid directory, got: %v", err)
	}
	if cleaner == nil {
		t.Error("Expected cleaner instance, got nil")
	}

	nonExistentPath := filepath.Join(safeTestRoot, "nonexistent")
	_, err = New(nonExistentPath)
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}

	symlinkPath := filepath.Join(safeTestRoot, "symlink")
	err = os.Symlink(safeTestRoot, symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	_, err = New(symlinkPath)
	if err == nil {
		t.Error("Expected error for symlink as working directory")
	}
	t.Logf("Got expected error for symlink: %v (type: %T)", err, err)
}

func TestValidatePathSecurity(t *testing.T) {
	safeTestRoot, cleanup := createSafeTestEnv(t)
	defer cleanup()

	cleaner, err := New(safeTestRoot)
	if err != nil {
		t.Fatalf("Failed to create cleaner: %v", err)
	}

	validPath := filepath.Join(safeTestRoot, "valid_dir")
	os.Mkdir(validPath, 0755)

	err = cleaner.validatePathSecurity(validPath)
	if err != nil {
		t.Errorf("Expected no error for valid path, got: %v", err)
	}

	traversalPath := filepath.Join(safeTestRoot, "..", "..", "fake_etc")
	err = cleaner.validatePathSecurity(traversalPath)
	if err == nil {
		t.Error("Expected error for path traversal attempt")
	}

	tempDir, _ := os.MkdirTemp("", "wdmt_outside_test")
	defer os.RemoveAll(tempDir)
	outsidePath := filepath.Join(tempDir, "outside")
	err = cleaner.validatePathSecurity(outsidePath)
	if err == nil {
		t.Error("Expected error for path outside working directory")
	}

	err = cleaner.validatePathSecurity(safeTestRoot)
	if err == nil {
		t.Error("Expected error for working directory itself")
	}

	nullBytePath := filepath.Join(safeTestRoot, "test\x00injection")
	err = cleaner.validatePathSecurity(nullBytePath)
	if err == nil {
		t.Error("Expected error for null byte injection")
	}
}

func TestSecureDeleteDirectory_SymlinkProtection(t *testing.T) {
	safeTestRoot, cleanup := createSafeTestEnv(t)
	defer cleanup()

	cleaner, err := New(safeTestRoot)
	if err != nil {
		t.Fatalf("Failed to create cleaner: %v", err)
	}

	safeTargetDir := filepath.Join(safeTestRoot, "safe_target")
	os.Mkdir(safeTargetDir, 0755)

	safeFakeSystemDir := filepath.Join(safeTestRoot, "fake_system_dir")
	os.Mkdir(safeFakeSystemDir, 0755)

	testFile := filepath.Join(safeFakeSystemDir, "important_file.txt")
	err = os.WriteFile(testFile, []byte("important data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	symlinkPath := filepath.Join(safeTestRoot, "malicious_symlink")
	err = os.Symlink(safeFakeSystemDir, symlinkPath)
	if err != nil {
		t.Fatalf("Failed to create test symlink: %v", err)
	}

	err = cleaner.secureDeleteDirectory(symlinkPath)
	if err == nil {
		t.Error("Expected error when trying to delete symlink")
	}
	if _, ok := err.(*SecurityError); !ok {
		t.Errorf("Expected SecurityError, got: %T", err)
	}

	if _, err := os.Lstat(symlinkPath); os.IsNotExist(err) {
		t.Error("Symlink was unexpectedly deleted")
	}

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Target file was unexpectedly deleted - symlink was followed!")
	}
}

func TestSecureRemoveAll_SymlinkHandling(t *testing.T) {
	safeTestRoot, cleanup := createSafeTestEnv(t)
	defer cleanup()

	cleaner, err := New(safeTestRoot)
	if err != nil {
		t.Fatalf("Failed to create cleaner: %v", err)
	}

	testDir := filepath.Join(safeTestRoot, "test_dir")
	os.Mkdir(testDir, 0755)

	regularFile := filepath.Join(testDir, "regular.txt")
	err = os.WriteFile(regularFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	subDir := filepath.Join(testDir, "subdir")
	os.Mkdir(subDir, 0755)

	symlinkInDir := filepath.Join(testDir, "internal_symlink")
	err = os.Symlink(regularFile, symlinkInDir)
	if err != nil {
		t.Fatalf("Failed to create internal symlink: %v", err)
	}

	externalDir := filepath.Join(safeTestRoot, "external")
	os.Mkdir(externalDir, 0755)

	importantFile := filepath.Join(externalDir, "important.txt")
	err = os.WriteFile(importantFile, []byte("critical data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create important file: %v", err)
	}

	maliciousSymlink := filepath.Join(testDir, "malicious")
	err = os.Symlink(externalDir, maliciousSymlink)
	if err != nil {
		t.Fatalf("Failed to create test symlink: %v", err)
	}

	err = cleaner.secureRemoveAll(testDir)
	if err != nil {
		t.Errorf("Failed to remove test directory: %v", err)
	}

	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("Test directory still exists")
	}

	if _, err := os.Stat(importantFile); os.IsNotExist(err) {
		t.Error("External file was unexpectedly deleted - symlink was followed!")
	}
}

func TestValidateTargets_ComprehensiveValidation(t *testing.T) {
	safeTestRoot, cleanup := createSafeTestEnv(t)
	defer cleanup()

	cleaner, err := New(safeTestRoot)
	if err != nil {
		t.Fatalf("Failed to create cleaner: %v", err)
	}

	validDir := filepath.Join(safeTestRoot, "valid")
	os.Mkdir(validDir, 0755)

	symlinkDir := filepath.Join(safeTestRoot, "symlink_dir")
	err = os.Symlink(validDir, symlinkDir)
	if err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	regularFile := filepath.Join(safeTestRoot, "regular.txt")
	err = os.WriteFile(regularFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	outsideTemp, err := os.MkdirTemp("", "wdmt_outside")
	if err != nil {
		t.Fatalf("Failed to create outside temp: %v", err)
	}
	defer os.RemoveAll(outsideTemp)
	outsidePath := filepath.Join(outsideTemp, "outside")

	targets := []scanner.CleanupTarget{
		{Path: validDir, Size: 1024, Type: "node_modules"},
		{Path: symlinkDir, Size: 512, Type: "dist"},
		{Path: regularFile, Size: 100, Type: "cache"},
		{Path: outsidePath, Size: 200, Type: "build"},
		{Path: filepath.Join(safeTestRoot, "nonexistent"), Size: 300, Type: "logs"},
	}

	validTargets, err := cleaner.ValidateTargets(targets)
	if err != nil {
		t.Errorf("Unexpected error during validation: %v", err)
	}

	if len(validTargets) != 1 {
		t.Errorf("Expected 1 valid target, got %d", len(validTargets))
	}

	if len(validTargets) > 0 && validTargets[0].Path != validDir {
		t.Errorf("Expected valid target to be %s, got %s", validDir, validTargets[0].Path)
	}
}

func TestPathTraversalProtection(t *testing.T) {
	safeTestRoot, cleanup := createSafeTestEnv(t)
	defer cleanup()

	cleaner, err := New(safeTestRoot)
	if err != nil {
		t.Fatalf("Failed to create cleaner: %v", err)
	}

	traversalAttempts := []string{
		filepath.Join(safeTestRoot, "..", "..", "..", "fake_passwd"),
		filepath.Join(safeTestRoot, "..", "..", "..", "fake_usr", "bin"),
		filepath.Join(safeTestRoot, "..", "..", "fake_home"),
		filepath.Join(safeTestRoot, "..", "sibling"),
		filepath.Join(safeTestRoot, "subdir", "..", "..", "fake_etc"),
	}

	for _, attempt := range traversalAttempts {
		t.Run(filepath.Base(attempt), func(t *testing.T) {
			err := cleaner.validatePathSecurity(attempt)
			if err == nil {
				t.Errorf("Expected error for traversal attempt: %s", attempt)
			}
		})
	}
}

func TestSecurityLogic_IsolatedFailureSimulation(t *testing.T) {
	safeTestRoot, cleanup := createSafeTestEnv(t)
	defer cleanup()

	fakeSystem := filepath.Join(safeTestRoot, "fake_system")
	fakeEtc := filepath.Join(fakeSystem, "etc")
	fakePasswd := filepath.Join(fakeEtc, "passwd")

	err := os.MkdirAll(fakeEtc, 0755)
	if err != nil {
		t.Fatalf("Failed to create fake system: %v", err)
	}

	err = os.WriteFile(fakePasswd, []byte("fake:passwd:data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create fake passwd: %v", err)
	}

	dangerousSymlink := filepath.Join(safeTestRoot, "fake_node_modules")
	err = os.Symlink(fakeEtc, dangerousSymlink)
	if err != nil {
		t.Fatalf("Failed to create dangerous symlink: %v", err)
	}

	cleaner, err := New(safeTestRoot)
	if err != nil {
		t.Fatalf("Failed to create cleaner: %v", err)
	}

	err = cleaner.secureDeleteDirectory(dangerousSymlink)
	if err == nil {
		t.Error("Security measures failed - symlink deletion was allowed!")
	}

	if _, err := os.Stat(fakePasswd); os.IsNotExist(err) {
		t.Error("Fake system was deleted - security measures failed!")
	}

	t.Logf("Security measures correctly prevented deletion of: %s", dangerousSymlink)
}

func TestSeparationOfConcerns_ScannerAndCleaner(t *testing.T) {
	safeTestRoot, cleanup := createSafeTestEnv(t)
	defer cleanup()

	safeDir := filepath.Join(safeTestRoot, "node_modules")
	os.Mkdir(safeDir, 0755)

	unsafeDir := filepath.Join(safeTestRoot, "unsafe_symlink")
	externalTarget := filepath.Join(safeTestRoot, "external_target")
	os.Mkdir(externalTarget, 0755)
	os.Symlink(externalTarget, unsafeDir)

	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(safeTestRoot)

	targets := []scanner.CleanupTarget{
		{
			Path:     safeDir,
			Name:     "node_modules",
			Size:     1024,
			Type:     "Node.js dependencies",
			Selected: true,
		},
		{
			Path:     unsafeDir,
			Name:     "unsafe_symlink",
			Size:     512,
			Type:     "Unknown",
			Selected: true,
		},
	}

	cleaner, err := New(safeTestRoot)
	if err != nil {
		t.Fatalf("Failed to create cleaner: %v", err)
	}

	validTargets, err := cleaner.ValidateTargets(targets)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if len(validTargets) != 1 {
		t.Errorf("Expected 1 valid target, got %d", len(validTargets))
	}

	if len(validTargets) > 0 && validTargets[0].Name != "node_modules" {
		t.Errorf("Expected safe target to be node_modules, got %s", validTargets[0].Name)
	}

	if len(validTargets) == len(targets) {
		t.Error("Cleaner should have filtered out unsafe targets")
	}
}
