package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileOperations(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	t.Run("CreateDir", func(t *testing.T) {
		path := filepath.Join(tmpDir, "test", "nested", "dir")
		if err := CreateDir(path); err != nil {
			t.Errorf("CreateDir() error = %v", err)
		}
		if !DirExists(path) {
			t.Error("Directory was not created")
		}
	})

	t.Run("WriteFile", func(t *testing.T) {
		path := filepath.Join(tmpDir, "test.txt")
		content := []byte("test content")
		if err := WriteFile(path, content); err != nil {
			t.Errorf("WriteFile() error = %v", err)
		}
		if !FileExists(path) {
			t.Error("File was not created")
		}

		// Verify content
		readContent, err := ReadFile(path)
		if err != nil {
			t.Errorf("ReadFile() error = %v", err)
		}
		if string(readContent) != "test content" {
			t.Errorf("ReadFile() = %s, want 'test content'", string(readContent))
		}
	})

	t.Run("IsEmptyDir", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty")
		os.MkdirAll(emptyDir, 0755)

		isEmpty, err := IsEmptyDir(emptyDir)
		if err != nil {
			t.Errorf("IsEmptyDir() error = %v", err)
		}
		if !isEmpty {
			t.Error("Expected directory to be empty")
		}
	})

	t.Run("ListDir", func(t *testing.T) {
		testDir := filepath.Join(tmpDir, "listtest")
		CreateDir(testDir)
		WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content1"))
		WriteFile(filepath.Join(testDir, "file2.txt"), []byte("content2"))

		entries, err := ListDir(testDir)
		if err != nil {
			t.Errorf("ListDir() error = %v", err)
		}
		if len(entries) != 2 {
			t.Errorf("ListDir() = %d entries, want 2", len(entries))
		}
	})

	t.Run("CopyFile", func(t *testing.T) {
		srcPath := filepath.Join(tmpDir, "source.txt")
		dstPath := filepath.Join(tmpDir, "destination.txt")

		WriteFile(srcPath, []byte("copy content"))
		if err := CopyFile(srcPath, dstPath); err != nil {
			t.Errorf("CopyFile() error = %v", err)
		}

		dstContent, _ := ReadFile(dstPath)
		if string(dstContent) != "copy content" {
			t.Error("Copied file has incorrect content")
		}
	})

	t.Run("FileExists", func(t *testing.T) {
		nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
		if FileExists(nonExistent) {
			t.Error("FileExists() returned true for non-existent file")
		}

		existent := filepath.Join(tmpDir, "existent.txt")
		WriteFile(existent, []byte("content"))
		if !FileExists(existent) {
			t.Error("FileExists() returned false for existing file")
		}
	})
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Ensure non-existent dir
	path := filepath.Join(tmpDir, "ensure", "test")
	if err := EnsureDir(path); err != nil {
		t.Errorf("EnsureDir() error = %v", err)
	}
	if !DirExists(path) {
		t.Error("Directory was not created by EnsureDir()")
	}

	// Ensure existing dir (should not error)
	if err := EnsureDir(path); err != nil {
		t.Errorf("EnsureDir() on existing dir error = %v", err)
	}
}
