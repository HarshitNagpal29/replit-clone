package fs

import (
	"os"
	"path/filepath"
)

type File struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// Fetches the contents of a directory
func FetchDir(dir string, baseDir string) ([]File, error) {
	var files []File

	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range dirEntries {
		fileType := "file"
		if entry.IsDir() {
			fileType = "dir"
		}
		files = append(files, File{
			Type: fileType,
			Name: entry.Name(),
			Path: filepath.Join(baseDir, entry.Name()),
		})
	}

	return files, nil
}

// Reads the contents of a file
func FetchFileContent(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Saves content to a file
func SaveFile(file string, content string) error {
	err := os.WriteFile(file, []byte(content), 0644)
	if err != nil {
		return err
	}
	return nil
}
