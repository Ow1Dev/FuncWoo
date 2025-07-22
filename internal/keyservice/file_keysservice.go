package keyservice

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSystemKeyService struct {
	filepath string
}

func NewFileSystemKeyService(filepath string) *FileSystemKeyService {
	return &FileSystemKeyService{
		filepath: filepath,
	}
}

func (f *FileSystemKeyService) GetKeyFromAction(action string) (string, error) {
	path := filepath.Join(f.filepath, action)
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open action file: %w", err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("error closing file %s: %v\n", path, err)
		}
	}()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading action file: %w", err)
	}
	return "", fmt.Errorf("action file %s is empty", path)
}
