package file

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func alloc(files []struct {
	Length int64
	Path   []string
}) (map[string]*os.File, error) {
	m := make(map[string]*os.File, len(files))
	for _, f := range files {
		if len(f.Path) == 0 {
			return nil, errors.New("alloc: file.Path == 0")
		}
		var filePath string
		if len(f.Path) > 1 {
			dirPath := filepath.Join(f.Path[:len(f.Path)-1]...)
			err := os.MkdirAll(dirPath, 0755)
			if err != nil {
				return nil, fmt.Errorf("alloc: %w", err)
			}
			filePath = strings.Join(f.Path, "/")
		} else {
			filePath = f.Path[0]
		}
		slog.Info(filePath)
		slog.Info(f.Path[0])
		file, err := os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("alloc: %w", err)
		}

		err = file.Truncate(int64(f.Length))
		if err != nil {
			return nil, fmt.Errorf("alloc: %w", err)
		}
		m[filePath] = file
	}

	return m, nil
}

func writeChunk(file *os.File, offset int64, data []byte) error {
	_, err := file.WriteAt(data, offset)
	return err
}

func delete(f *os.File) error {
	name := f.Name()
	_ = f.Close()
	return os.Remove(name)
}
