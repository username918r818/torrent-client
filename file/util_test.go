package file_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/username918r818/torrent-client/file"
)

func TestAlloc(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("successful file creation", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "file1.txt")

		files := []struct {
			Length uint64
			Path   []string
		}{
			{Length: 1024, Path: []string{tempDir, "file1.txt"}},
		}

		_, err := file.Alloc(files)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		_, err = os.Stat(filePath)
		if err != nil {
			t.Fatalf("expected file to be created, but got error: %v", err)
		}
	})
	t.Run("empty path", func(t *testing.T) {
		files := []struct {
			Length uint64
			Path   []string
		}{
			{Length: 1024, Path: []string{}},
		}

		_, err := file.Alloc(files)
		if err == nil {
			t.Fatal("expected error, but got none")
		}
		if err.Error() != "Alloc: file.Path == 0" {
			t.Fatalf("expected error message 'Alloc: file.Path == 0', got %v", err)
		}
	})

	t.Run("directory creation error", func(t *testing.T) {
		files := []struct {
			Length uint64
			Path   []string
		}{
			{Length: 1024, Path: []string{"/invalid:dir", "file1.txt"}},
		}

		_, err := file.Alloc(files)
		if err == nil {
			t.Fatal("expected error, but got none")
		}
	})
}

func TestWriteChunk(t *testing.T) {
	t.Run("successful write", func(t *testing.T) {
		f, err := os.CreateTemp("", "testfile-")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())

		data := []byte("Hello, world!")
		offset := int64(0)

		err = file.WriteChunk(f, offset, data)
		if err != nil {
			t.Fatalf("expected no error, but got %v", err)
		}

		f.Seek(0, 0)
		buf := make([]byte, len(data))
		_, err = f.Read(buf)
		if err != nil {
			t.Fatalf("failed to read from file: %v", err)
		}

		if !bytes.Equal(buf, data) {
			t.Fatalf("expected data %s, got %s", data, buf)
		}
	})

	t.Run("write error", func(t *testing.T) {
		f, err := os.CreateTemp("", "testfile-")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		defer os.Remove(f.Name())

		f.Close()

		data := []byte("Hello, world!")
		err = file.WriteChunk(f, 0, data)
		if err == nil {
			t.Fatal("expected error, but got none")
		}
	})
}
