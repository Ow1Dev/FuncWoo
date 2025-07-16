package keyservice

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileSystemKeyService struct{}

func NewFileSystemKeyService() *FileSystemKeyService {
	return &FileSystemKeyService{}
}

func (f *FileSystemKeyService) GetKeyFromAction(action string) (string, error) {
	path := filepath.Join("/var/lib/noctifunc/action", action)
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open action file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading action file: %w", err)
	}
	return "", fmt.Errorf("action file %s is empty", path)
}
