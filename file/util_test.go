package file

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestAlloc(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("successful file creation", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "file1.txt")

		files := []struct {
			Length int64
			Path   []string
		}{
			{Length: 1024, Path: []string{tempDir, "file1.txt"}},
		}

		_, err := alloc(files)
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
			Length int64
			Path   []string
		}{
			{Length: 1024, Path: []string{}},
		}

		_, err := alloc(files)
		if err == nil {
			t.Fatal("expected error, but got none")
		}
		if err.Error() != "alloc: Path == 0" {
			t.Fatalf("expected error message 'alloc: Path == 0', got %v", err)
		}
	})

	t.Run("directory creation error", func(t *testing.T) {
		files := []struct {
			Length int64
			Path   []string
		}{
			{Length: 1024, Path: []string{"/invalid:dir", "file1.txt"}},
		}

		_, err := alloc(files)
		if err == nil {
			t.Fatal("expected error, but got none")
		}
	})

	t.Run("inner directories with files", func(t *testing.T) {
		nestedDir := filepath.Join(tempDir, "dir1", "dir2")
		filePath := filepath.Join(nestedDir, "file1.txt")

		files := []struct {
			Length int64
			Path   []string
		}{
			{Length: 1024, Path: []string{nestedDir, "file1.txt"}},
		}

		_, err := alloc(files)
		if err != nil {
			t.Fatalf("expected no error, but got %v", err)
		}

		if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
			t.Fatalf("expected directory to be created, but got error: %v", err)
		}

		_, err = os.Stat(filePath)
		if err != nil {
			t.Fatalf("expected file to be created, but got error: %v", err)
		}

		t.Run("multiple files with different levels of nesting", func(t *testing.T) {
			nestedDirs := []struct {
				Path string
				File string
			}{
				{Path: filepath.Join(tempDir, "dir1"), File: "file1.txt"},
				{Path: filepath.Join(tempDir, "dir1", "subdir1"), File: "file2.txt"},
				{Path: filepath.Join(tempDir, "dir2"), File: "file3.txt"},
				{Path: filepath.Join(tempDir, "dir2", "subdir2", "subsubdir"), File: "file4.txt"},
			}

			files := []struct {
				Length int64
				Path   []string
			}{
				{Length: 1024, Path: []string{nestedDirs[0].Path, nestedDirs[0].File}},
				{Length: 1024, Path: []string{nestedDirs[1].Path, nestedDirs[1].File}},
				{Length: 1024, Path: []string{nestedDirs[2].Path, nestedDirs[2].File}},
				{Length: 1024, Path: []string{nestedDirs[3].Path, nestedDirs[3].File}},
			}

			_, err := alloc(files)
			if err != nil {
				t.Fatalf("expected no error, but got %v", err)
			}

			for _, dir := range nestedDirs {
				if _, err := os.Stat(dir.Path); os.IsNotExist(err) {
					t.Fatalf("expected directory %s to be created, but got error: %v", dir.Path, err)
				}

				filePath := filepath.Join(dir.Path, dir.File)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Fatalf("expected file %s to be created, but got error: %v", filePath, err)
				}
			}
		})

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

		err = writeChunk(f, offset, data)
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
		err = writeChunk(f, 0, data)
		if err == nil {
			t.Fatal("expected error, but got none")
		}
	})
}
