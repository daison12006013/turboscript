/*
 * TurboScript - File system helper functions for storage backends
 */

package fileupload

import (
	"mime"
	"os"
	"path/filepath"
)

// Helper functions for local filesystem operations

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0750)
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path) // #nosec G304 - This is a controlled file read operation within the file upload plugin
}

func removeFile(path string) error {
	return os.Remove(path)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getFileStat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func getContentType(filename string) string {
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}
