// Package main provides MIME type testing utilities for TurboScript.
package main

import (
	"fmt"
	"mime"
	"path/filepath"
)

func main() {
	fmt.Println("MIME type for .txt:", mime.TypeByExtension(".txt"))
	fmt.Println("MIME type for test.txt:", mime.TypeByExtension(filepath.Ext("test.txt")))

	// Test various extensions
	extensions := []string{".txt", ".json", ".html", ".css", ".js", ".png", ".jpg"}
	for _, ext := range extensions {
		mimeType := mime.TypeByExtension(ext)
		fmt.Printf("Extension %s -> MIME type: %s\n", ext, mimeType)
	}
}
