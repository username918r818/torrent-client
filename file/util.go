package file

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func Alloc(files []struct {
	Length int64
	Path   []string
}) (map[string]*os.File, error) {
	m := make(map[string]*os.File, len(files))
	for _, f := range files {
		if len(f.Path) == 0 {
			return nil, errors.New("Alloc: file.Path == 0")
		}
		var filepath string
		if len(f.Path) > 1 {
			dirPath := strings.Join(f.Path[:len(f.Path)-1], "/")
			err := os.MkdirAll(dirPath, 0744)
			if err != nil {
				return nil, fmt.Errorf("Alloc: %w", err)
			}
			filepath = strings.Join(f.Path, "/")
		} else {
			filepath = f.Path[0]
		}
		slog.Info(filepath)
		slog.Info(f.Path[0])
		file, err := os.Create(filepath)
		if err != nil {
			return nil, fmt.Errorf("Alloc: %w", err)
		}

		err = file.Truncate(int64(f.Length))
		if err != nil {
			return nil, fmt.Errorf("Alloc: %w", err)
		}
		m[filepath] = file
	}

	return m, nil
}

func WriteChunk(file *os.File, offset int64, data []byte) error {
	_, err := file.WriteAt(data, offset)
	return err
}
