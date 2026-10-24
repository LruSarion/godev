package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotContentAndExclusions(t *testing.T) {
	root := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	read := func() snapshot {
		t.Helper()
		s, err := scan(context.Background(), root)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	write("main.go", "aaa")
	write("go.mod", "module example")
	write("go.sum", "sum")
	write("page.html", "ignored")
	write(".local/generated.go", "ignored")
	write("vendor/lib/main.go", "ignored")
	before := read()
	if len(before) != 3 {
		t.Fatalf("tracked %d files", len(before))
	}
	info, err := os.Stat(filepath.Join(root, "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	write("main.go", "bbb")
	if err := os.Chtimes(filepath.Join(root, "main.go"), info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	after := read()
	if same(before, after) {
		t.Fatal("same-size content edit with unchanged mtime missed")
	}
	write("page.html", "still ignored")
	if !same(after, read()) {
		t.Fatal("untracked file changed snapshot")
	}
	if err := os.Remove(filepath.Join(root, "main.go")); err != nil {
		t.Fatal(err)
	}
	if same(after, read()) {
		t.Fatal("deletion missed")
	}
	write("new.go", "new")
	if same(after, read()) {
		t.Fatal("rename/addition missed")
	}
}

func TestSnapshotCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := scan(ctx, t.TempDir()); err != context.Canceled {
		t.Fatalf("got %v", err)
	}
}

func TestSnapshotDoesNotConfuseMissingZeroDigest(t *testing.T) {
	a := snapshot{"a.go": {}}
	b := snapshot{"b.go": {}}
	if same(a, b) {
		t.Fatal("different paths treated as equal")
	}
}
