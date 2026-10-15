package syncer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// ComputeSHA256 returns the hexadecimal SHA-256 checksum of the file at filePath.
func ComputeSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// IsIdentical checks whether two file system paths (files or directories)
// have identical content by comparing sizes and SHA-256 hashes recursively.
func IsIdentical(pathA, pathB string) (bool, error) {
	statA, err := os.Stat(pathA)
	if err != nil {
		return false, err
	}
	statB, err := os.Stat(pathB)
	if err != nil {
		return false, err
	}

	if statA.IsDir() != statB.IsDir() {
		return false, nil
	}
	if !statA.IsDir() {
		if statA.Size() != statB.Size() {
			return false, nil
		}
		hashA, err := ComputeSHA256(pathA)
		if err != nil {
			return false, err
		}
		hashB, err := ComputeSHA256(pathB)
		if err != nil {
			return false, err
		}
		return hashA == hashB, nil
	}

	// For directories, compare all files and subdirectories recursively
	dirsA := make(map[string]bool)
	filesA := make(map[string]os.FileInfo)
	_ = filepath.Walk(pathA, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(pathA, p)
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			dirsA[rel] = true
		} else {
			filesA[rel] = info
		}
		return nil
	})

	dirsB := make(map[string]bool)
	filesB := make(map[string]os.FileInfo)
	_ = filepath.Walk(pathB, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(pathB, p)
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			dirsB[rel] = true
		} else {
			filesB[rel] = info
		}
		return nil
	})

	if len(dirsA) != len(dirsB) || len(filesA) != len(filesB) {
		return false, nil
	}

	for rel := range dirsA {
		if !dirsB[rel] {
			return false, nil
		}
	}

	for rel, infoA := range filesA {
		infoB, ok := filesB[rel]
		if !ok || infoA.Size() != infoB.Size() {
			return false, nil
		}
		hA, err := ComputeSHA256(filepath.Join(pathA, rel))
		if err != nil {
			return false, err
		}
		hB, err := ComputeSHA256(filepath.Join(pathB, rel))
		if err != nil {
			return false, err
		}
		if hA != hB {
			return false, nil
		}
	}

	return true, nil
}

// CopyFile copies a single file from src to dst, creating any necessary parent directories.
// It guards against truncating src when src and dst resolve to the same physical file,
// and ensures dstFile.Close() errors are caught and returned.
func CopyFile(src, dst string) (err error) {
	cleanSrc := filepath.Clean(src)
	cleanDst := filepath.Clean(dst)
	if cleanSrc == cleanDst {
		return nil
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// Guard against truncating same physical file (rel vs abs, hardlinks, symlinks)
	if dstInfo, err := os.Stat(dst); err == nil {
		if os.SameFile(srcInfo, dstInfo) {
			return nil
		}
	}

	perm := srcInfo.Mode().Perm()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := dstFile.Close()
		if err == nil {
			err = closeErr
		}
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Update permissions in case dst file already existed
	_ = os.Chmod(dst, perm)

	return nil
}

// CopyDir recursively copies the directory tree at src to dst.
func CopyDir(src, dst string) error {
	cleanSrc := filepath.Clean(src)
	cleanDst := filepath.Clean(dst)
	if cleanSrc == cleanDst {
		return nil
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if dstInfo, err := os.Stat(dst); err == nil {
		if os.SameFile(srcInfo, dstInfo) {
			return nil
		}
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// CopyEntry copies a file or directory tree from src to dst.
func CopyEntry(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return CopyDir(src, dst)
	}
	return CopyFile(src, dst)
}
