package main

import (
	"context"
	"crypto/sha256"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type snapshot map[string][sha256.Size]byte

func isOutside(rel string) bool { return strings.HasPrefix(rel, ".."+string(filepath.Separator)) }

func excluded(name string) bool {
	switch name {
	case ".git", ".local", ".idea", ".vscode", "vendor", "node_modules", "dist", "build", "tmp", "coverage":
		return true
	}
	return false
}

func scan(ctx context.Context, root string) (snapshot, error) {
	result := make(snapshot)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && excluded(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") && name != "go.mod" && name != "go.sum" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, readErr := io.Copy(hash, file)
		closeErr := file.Close()
		if readErr != nil {
			return readErr
		}
		if closeErr != nil {
			return closeErr
		}
		var digest [sha256.Size]byte
		copy(digest[:], hash.Sum(nil))
		result[path] = digest
		return nil
	})
	return result, err
}

func same(a, b snapshot) bool {
	if len(a) != len(b) {
		return false
	}
	for path, digest := range a {
		other, ok := b[path]
		if !ok || other != digest {
			return false
		}
	}
	return true
}
