/*
 * TurboScript - A hybrid web framework combining TypeScript and Go
 *
 * Copyright (c) 2025 TurboScript Project Contributors
 * Author: Daison Cariño <daison12006013@gmail.com>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Based on TurboScript: https://github.com/daison12006013/turboscript
 */

// Package tsengine provides file resolution utilities for TypeScript/JavaScript files.
package tsengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daison12006013/turboscript/internal/logger"
)

// File extension constants.
const (
	tsExtension = ".ts"
	jsExtension = ".js"
)

// FileResolver provides intelligent file resolution for TypeScript/JavaScript files.
//
// It automatically resolves file paths by checking for both .ts and .js extensions,
// with configurable preference for development (.ts) or production (.js) environments.
type FileResolver struct {
	preferJS bool // When true, prefer .js files over .ts files
}

// NewFileResolver creates a new file resolver instance.
//
// Parameters:
//   - preferJS: If true, prefer .js files over .ts files when both exist
func NewFileResolver(preferJS bool) *FileResolver {
	return &FileResolver{
		preferJS: preferJS,
	}
}

// ResolveFile intelligently resolves a file path to the correct TypeScript or JavaScript file.
//
// Resolution Logic:
//  1. If the provided path already has a .ts or .js extension, check if that file exists
//  2. If not found or no extension provided, try both .ts and .js extensions
//  3. Apply preference (development prefers .ts, production prefers .js)
//  4. Return the first existing file found
//
// Parameters:
//   - filePath: The file path to resolve (with or without extension)
//
// Returns:
//   - The resolved absolute file path
//   - Error if no suitable file is found
func (fr *FileResolver) ResolveFile(filePath string) (string, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
	}

	logger.Debug("Resolving file: %s (prefer JS: %v)", absPath, fr.preferJS)

	// Check if the file already has a .ts or .js extension
	ext := strings.ToLower(filepath.Ext(absPath))
	if ext == tsExtension || ext == jsExtension {
		// File has an extension, check if it exists
		if _, err := os.Stat(absPath); err == nil {
			logger.Debug("Found exact file: %s", absPath)
			return absPath, nil
		}

		// File with specified extension doesn't exist, try the alternative
		basePath := strings.TrimSuffix(absPath, ext)
		alternativeExt := jsExtension
		if ext == jsExtension {
			alternativeExt = tsExtension
		}

		alternativePath := basePath + alternativeExt
		if _, err := os.Stat(alternativePath); err == nil {
			logger.Debug("Found alternative file: %s (instead of %s)", alternativePath, absPath)
			return alternativePath, nil
		}

		// Neither the specified nor alternative extension exists
		return "", fmt.Errorf("file not found: %s (also tried %s)", absPath, alternativePath)
	}

	// No extension provided, try both extensions with preference
	basePath := absPath

	var primaryPath, secondaryPath string
	if fr.preferJS {
		primaryPath = basePath + jsExtension
		secondaryPath = basePath + tsExtension
	} else {
		primaryPath = basePath + tsExtension
		secondaryPath = basePath + jsExtension
	}

	// Try primary extension first
	if _, err := os.Stat(primaryPath); err == nil {
		logger.Debug("Found preferred file: %s", primaryPath)
		return primaryPath, nil
	}

	// Try secondary extension
	if _, err := os.Stat(secondaryPath); err == nil {
		logger.Debug("Found fallback file: %s", secondaryPath)
		return secondaryPath, nil
	}

	// No files found with either extension
	return "", fmt.Errorf("file not found: %s (tried %s and %s)", filePath, primaryPath, secondaryPath)
}

// ResolveFileQuiet is like ResolveFile but only logs at debug level to avoid noise.
func (fr *FileResolver) ResolveFileQuiet(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", filePath, err)
	}

	// Check if the file already has a .ts or .js extension
	ext := strings.ToLower(filepath.Ext(absPath))
	if ext == tsExtension || ext == jsExtension {
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}

		basePath := strings.TrimSuffix(absPath, ext)
		alternativeExt := jsExtension
		if ext == jsExtension {
			alternativeExt = tsExtension
		}

		alternativePath := basePath + alternativeExt
		if _, err := os.Stat(alternativePath); err == nil {
			return alternativePath, nil
		}

		return "", fmt.Errorf("file not found: %s (also tried %s)", absPath, alternativePath)
	}

	// No extension provided, try both extensions with preference
	basePath := absPath

	var primaryPath, secondaryPath string
	if fr.preferJS {
		primaryPath = basePath + jsExtension
		secondaryPath = basePath + tsExtension
	} else {
		primaryPath = basePath + tsExtension
		secondaryPath = basePath + jsExtension
	}

	if _, err := os.Stat(primaryPath); err == nil {
		return primaryPath, nil
	}

	if _, err := os.Stat(secondaryPath); err == nil {
		return secondaryPath, nil
	}

	return "", fmt.Errorf("file not found: %s (tried %s and %s)", filePath, primaryPath, secondaryPath)
}

// GetRecommendedResolver creates a file resolver with recommended settings based on environment.
//
// Development: Prefers .ts files (for hot-reloading and debugging).
// Production: Prefers .js files (for compiled distributions).
func GetRecommendedResolver() *FileResolver {
	// Check various indicators of production environment
	isProduction := false

	// Method 1: Check working directory for dist/ pattern
	if cwd, err := os.Getwd(); err == nil {
		if strings.Contains(cwd, "dist") || strings.HasSuffix(cwd, "dist") {
			isProduction = true
		}
	}

	// Method 2: Check if app/ directory contains .js files (indicating compilation)
	if !isProduction {
		if appDir := "app"; dirExists(appDir) {
			if hasJSFiles := directoryContainsJS(appDir); hasJSFiles {
				isProduction = true
			}
		}
	}

	// Method 3: Check environment variable
	if !isProduction {
		if env := os.Getenv("NODE_ENV"); env == "production" {
			isProduction = true
		}
		if env := os.Getenv("APP_ENV"); env == "production" {
			isProduction = true
		}
	}

	resolver := NewFileResolver(isProduction)

	if isProduction {
		logger.Debug("File resolver configured for production (prefers .js files)")
	} else {
		logger.Debug("File resolver configured for development (prefers .ts files)")
	}

	return resolver
}

// GetResolverFromConfig creates a file resolver with settings from configuration.
//
// The configuration can specify either prefer_ts or prefer_js. If prefer_ts is true,
// it takes precedence over prefer_js. If neither is set, it falls back to environment detection.
func GetResolverFromConfig(preferTS, preferJS bool) *FileResolver {
	// prefer_ts takes precedence over prefer_js
	if preferTS {
		logger.Debug("File resolver configured from config (prefers .ts files)")
		return NewFileResolver(false) // false means prefer .ts files
	}

	if preferJS {
		logger.Debug("File resolver configured from config (prefers .js files)")
		return NewFileResolver(true) // true means prefer .js files
	}

	// Fall back to environment detection if no config preference is set
	logger.Debug("File resolver using environment detection")
	return GetRecommendedResolver()
}

// Helper function to check if directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// Helper function to check if directory contains any .js files.
func directoryContainsJS(dirPath string) bool {
	found := false
	_ = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(path), jsExtension) {
			found = true
			return fmt.Errorf("found js file") // Use error to break early
		}
		return nil
	})
	return found
}
