package syncer

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileAndHash(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcFile := filepath.Join(srcDir, "test.txt")
	dstFile := filepath.Join(dstDir, "test.txt")

	content := []byte("hello dotsaver\n")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to write src: %v", err)
	}

	if err := CopyFile(srcFile, dstFile); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	same, err := IsIdentical(srcFile, dstFile)
	if err != nil || !same {
		t.Fatalf("Expected files to be identical, err: %v, same: %v", err, same)
	}
}

func TestCopyDirRecursive(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	subDir := filepath.Join(srcDir, "sub", "deep")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to mkdir: %v", err)
	}
	testFile := filepath.Join(subDir, "init.lua")
	if err := os.WriteFile(testFile, []byte("vim.opt.number = true"), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	targetDst := filepath.Join(dstDir, "copied_tree")
	if err := CopyDir(srcDir, targetDst); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	destFile := filepath.Join(targetDst, "sub", "deep", "init.lua")
	same, err := IsIdentical(testFile, destFile)
	if err != nil || !same {
		t.Fatalf("Copied directory file mismatch: %v, same: %v", err, same)
	}

	// Verify whole directory identical
	dirSame, err := IsIdentical(srcDir, targetDst)
	if err != nil || !dirSame {
		t.Fatalf("Copied directory tree mismatch: %v, same: %v", err, dirSame)
	}
}

func TestComputeSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "sha_test.txt")
	content := []byte("some deterministic payload for hashing")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	expectedHashBytes := sha256.Sum256(content)
	expectedHash := hex.EncodeToString(expectedHashBytes[:])

	actualHash, err := ComputeSHA256(filePath)
	if err != nil {
		t.Fatalf("ComputeSHA256 failed: %v", err)
	}

	if actualHash != expectedHash {
		t.Errorf("Expected sha256 %s, got %s", expectedHash, actualHash)
	}

	// Non-existent file should error
	_, err = ComputeSHA256(filepath.Join(tmpDir, "nonexistent.txt"))
	if err == nil {
		t.Fatalf("Expected error for non-existent file, got nil")
	}
}

func TestCopyEntry(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Test single file CopyEntry
	srcFile := filepath.Join(srcDir, "single.conf")
	dstFile := filepath.Join(dstDir, "single.conf")
	if err := os.WriteFile(srcFile, []byte("setting=on"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := CopyEntry(srcFile, dstFile); err != nil {
		t.Fatalf("CopyEntry for file failed: %v", err)
	}

	same, err := IsIdentical(srcFile, dstFile)
	if err != nil || !same {
		t.Fatalf("CopyEntry file mismatch: %v, same: %v", err, same)
	}

	// Test directory CopyEntry
	nestedSrc := filepath.Join(srcDir, "folder")
	if err := os.MkdirAll(nestedSrc, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedSrc, "item.txt"), []byte("item"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	nestedDst := filepath.Join(dstDir, "folder_copy")
	if err := CopyEntry(nestedSrc, nestedDst); err != nil {
		t.Fatalf("CopyEntry for dir failed: %v", err)
	}

	dirSame, err := IsIdentical(nestedSrc, nestedDst)
	if err != nil || !dirSame {
		t.Fatalf("CopyEntry dir mismatch: %v, same: %v", err, dirSame)
	}
}

func TestIsIdentical(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "f1.txt")
	file2 := filepath.Join(tmpDir, "f2.txt")
	file3 := filepath.Join(tmpDir, "f3.txt")

	if err := os.WriteFile(file1, []byte("abcdef"), 0644); err != nil {
		t.Fatal(err)
	}
	// file2 has same length, different content
	if err := os.WriteFile(file2, []byte("abcdeg"), 0644); err != nil {
		t.Fatal(err)
	}
	// file3 has different length
	if err := os.WriteFile(file3, []byte("abcdef_long"), 0644); err != nil {
		t.Fatal(err)
	}

	// Same length, different content -> false
	diffContent, err := IsIdentical(file1, file2)
	if err != nil {
		t.Fatalf("IsIdentical failed: %v", err)
	}
	if diffContent {
		t.Errorf("Expected IsIdentical to be false for differing content")
	}

	// Different length -> false
	diffLength, err := IsIdentical(file1, file3)
	if err != nil {
		t.Fatalf("IsIdentical failed: %v", err)
	}
	if diffLength {
		t.Errorf("Expected IsIdentical to be false for differing length")
	}

	// File vs Dir -> false
	dir1 := filepath.Join(tmpDir, "d1")
	if err := os.MkdirAll(dir1, 0755); err != nil {
		t.Fatal(err)
	}
	fileVsDir, err := IsIdentical(file1, dir1)
	if err != nil {
		t.Fatalf("IsIdentical failed: %v", err)
	}
	if fileVsDir {
		t.Errorf("Expected IsIdentical to be false for file vs dir")
	}

	// Non-existent path -> error
	_, err = IsIdentical(file1, filepath.Join(tmpDir, "missing.txt"))
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}

	// Dir comparison with differing files
	dir2 := filepath.Join(tmpDir, "d2")
	if err := os.MkdirAll(dir2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir1, "config.json"), []byte(`{"a": 1}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir2, "config.json"), []byte(`{"a": 2}`), 0644); err != nil {
		t.Fatal(err)
	}

	dirSame, err := IsIdentical(dir1, dir2)
	if err != nil {
		t.Fatalf("IsIdentical on dirs failed: %v", err)
	}
	if dirSame {
		t.Errorf("Expected IsIdentical to be false for dirs with different file content")
	}

	// Dir comparison with differing file count
	if err := os.WriteFile(filepath.Join(dir2, "extra.txt"), []byte("extra"), 0644); err != nil {
		t.Fatal(err)
	}
	dirCountMismatch, err := IsIdentical(dir1, dir2)
	if err != nil {
		t.Fatalf("IsIdentical on dirs failed: %v", err)
	}
	if dirCountMismatch {
		t.Errorf("Expected IsIdentical to be false for dirs with differing file counts")
	}
}

func TestCopyErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// CopyFile with non-existent source
	err := CopyFile(filepath.Join(tmpDir, "nonexistent.txt"), filepath.Join(tmpDir, "dest.txt"))
	if err == nil {
		t.Errorf("Expected error copying non-existent file, got nil")
	}

	// CopyDir with non-existent source
	err = CopyDir(filepath.Join(tmpDir, "nonexistent_dir"), filepath.Join(tmpDir, "dest_dir"))
	if err == nil {
		t.Errorf("Expected error copying non-existent dir, got nil")
	}

	// CopyEntry with non-existent source
	err = CopyEntry(filepath.Join(tmpDir, "nonexistent_entry"), filepath.Join(tmpDir, "dest_entry"))
	if err == nil {
		t.Errorf("Expected error copying non-existent entry, got nil")
	}
}

func TestCopySameFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "same.txt")
	content := []byte("important content that should not be truncated")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Identical paths
	if err := CopyFile(filePath, filePath); err != nil {
		t.Fatalf("CopyFile on identical paths returned error: %v", err)
	}

	readBack, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(readBack) != string(content) {
		t.Fatalf("File was corrupted/truncated! Expected %q, got %q", string(content), string(readBack))
	}

	// 2. Relative vs absolute path resolving to same file
	relPath := filepath.Join(".", "testdata_same.tmp")
	if err := os.WriteFile(relPath, content, 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(relPath)

	absPath, err := filepath.Abs(relPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(relPath, absPath); err != nil {
		t.Fatalf("CopyFile on same file via rel/abs returned error: %v", err)
	}

	readBackRel, err := os.ReadFile(relPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(readBackRel) != string(content) {
		t.Fatalf("File was corrupted/truncated via rel/abs! Expected %q, got %q", string(content), string(readBackRel))
	}
}

func TestCopySameDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "folder")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "f.txt"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := CopyDir(nestedDir, nestedDir); err != nil {
		t.Fatalf("CopyDir on same dir failed: %v", err)
	}
}

func TestIsIdenticalEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	dirA := filepath.Join(tmpDir, "dirA")
	dirB := filepath.Join(tmpDir, "dirB")

	if err := os.MkdirAll(filepath.Join(dirA, "empty_subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dirB, 0755); err != nil {
		t.Fatal(err)
	}

	// dirA has an empty subdir, dirB has none. They should NOT be identical.
	same, err := IsIdentical(dirA, dirB)
	if err != nil {
		t.Fatalf("IsIdentical failed: %v", err)
	}
	if same {
		t.Errorf("Expected IsIdentical to be false when one dir has empty subdir")
	}
}
