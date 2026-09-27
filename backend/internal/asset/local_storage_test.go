package asset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStorageLiteralObjectKeys(t *testing.T) {
	for _, name := range []string{"lesson.pdf", "Lesson%20One.pdf", "Lesson%25One.pdf", "Lesson%2FOne.pdf", "100%.pdf"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			dir := "team-agp-resources/objects/0000000000000000000000000000000a"
			key := dir + "/" + name
			content := []byte("migrated literal file " + name)
			if err := os.MkdirAll(filepath.Join(root, dir), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, key), content, 0o600); err != nil {
				t.Fatal(err)
			}
			storage := NewLocalStorage(root)
			resolved, err := storage.Resolve(t.Context(), key)
			if err != nil {
				t.Fatalf("resolve migrated object: %v", err)
			}
			data, err := os.ReadFile(resolved.AbsolutePath)
			if err != nil || string(data) != string(content) || resolved.OriginalName != name {
				t.Fatalf("resolved wrong object: %+v bytes=%q err=%v", resolved, data, err)
			}
			stored, err := storage.Save(t.Context(), dir, name, strings.NewReader(string(content)))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(content)
			if stored.StoragePath != key || stored.ChecksumSHA256 != hex.EncodeToString(sum[:]) || stored.FileSize != uint64(len(content)) {
				t.Fatalf("stored object metadata changed: %+v", stored)
			}
			if err := storage.Delete(t.Context(), key); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, key)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("literal object remains after delete: %v", err)
			}
		})
	}
}

func TestLocalStorageSaveRemovesPartialFile(t *testing.T) {
	root := t.TempDir()
	storage := NewLocalStorage(root)

	_, err := storage.Save(context.Background(), "team-agp-resources/objects/00000000000000000000000000000001", "partial.bin", &failingReader{})
	if err == nil {
		t.Fatal("Save succeeded, want copy error")
	}

	var files []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk storage root: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("partial files = %v, want none", files)
	}
}

func TestLocalStorageRejectsUnmanagedPath(t *testing.T) {
	t.Parallel()

	storage := NewLocalStorage(t.TempDir())
	if _, err := storage.Save(context.Background(), "unmanaged", "lesson.pdf", strings.NewReader("pdf")); err == nil {
		t.Fatal("Save accepted an unmanaged resource path")
	}
	if _, err := storage.Resolve(context.Background(), "unmanaged/lesson.pdf"); err == nil {
		t.Fatal("Resolve accepted an unmanaged resource path")
	}
	if err := storage.Delete(context.Background(), "unmanaged/lesson.pdf"); err == nil {
		t.Fatal("Delete accepted an unmanaged resource path")
	}
}

func TestLocalStorageSaveUsesStableResourcePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storage := NewLocalStorage(root)
	object, err := storage.Save(
		context.Background(),
		"team-agp-resources/objects/0000000000000000000000000000000a",
		"lesson.pdf",
		strings.NewReader("version two"),
	)
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	want := "team-agp-resources/objects/0000000000000000000000000000000a/lesson.pdf"
	if object.StoragePath != want {
		t.Fatalf("storage path = %q, want %q", object.StoragePath, want)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(want))); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}
}

func TestLocalStorageResolveRejectsSymlink(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	storage := NewLocalStorage(root)
	if _, err := storage.Resolve(context.Background(), "linked/secret.txt"); err == nil {
		t.Fatal("Resolve accepted a path containing a symlink")
	}
	if err := storage.Delete(context.Background(), "linked/secret.txt"); err == nil {
		t.Fatal("Delete accepted a path containing a symlink")
	}
	if _, err := os.Stat(filepath.Join(outside, "secret.txt")); err != nil {
		t.Fatalf("outside file was removed: %v", err)
	}
}

type failingReader struct {
	read bool
}

func (r *failingReader) Read(buffer []byte) (int, error) {
	if !r.read {
		r.read = true
		return copy(buffer, "partial"), nil
	}
	return 0, errors.New("read failed")
}

var _ io.Reader = (*failingReader)(nil)
