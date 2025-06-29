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

// Package tsengine provides compiled JavaScript caching utilities.
package tsengine

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
)

// CacheUtils manages compiled JavaScript code caching.
type CacheUtils struct {
	cache    sync.Map
	compiler *CompilerUtils
}

// NewCacheUtils creates a new cache utilities instance.
func NewCacheUtils(compiler *CompilerUtils) *CacheUtils {
	return &CacheUtils{
		compiler: compiler,
	}
}

// GetCompiledJS gets compiled JavaScript with caching.
func (cu *CacheUtils) GetCompiledJS(tsPath string) (string, error) {
	// Get file info for cache validation
	absPath, err := filepath.Abs(tsPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Check cache
	if cached, ok := cu.cache.Load(absPath); ok {
		if cachedJS, ok := cached.(*CachedJS); ok {
			// Validate cache - check if file has been modified
			if cachedJS.ModTime.Equal(fileInfo.ModTime()) && cachedJS.FileSize == fileInfo.Size() {
				logger.Debug("Using cached JavaScript compilation for %s", tsPath)
				return cachedJS.Code, nil
			}
		}
	}

	// Not in cache or invalid - compile
	logger.Debug("Compiling TypeScript (cache miss): %s", tsPath)
	startTime := time.Now()
	jsCode, err := cu.compiler.ConvertTSToJS(tsPath)
	if err != nil {
		return "", err
	}
	compilationTime := time.Since(startTime)

	// Log slow compilations that might cause cold start issues
	if compilationTime > 100*time.Millisecond {
		logger.Debug("TypeScript compilation took %.2f seconds for %s", compilationTime.Seconds(), tsPath)
	}

	// Store in cache
	cu.cache.Store(absPath, &CachedJS{
		Code:      jsCode,
		Timestamp: time.Now(),
		FileSize:  fileInfo.Size(),
		ModTime:   fileInfo.ModTime(),
	})

	return jsCode, nil
}

// ClearOldCache clears compilation cache entries older than the specified duration.
func (cu *CacheUtils) ClearOldCache(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)

	cu.cache.Range(func(key, value any) bool {
		if cachedJS, ok := value.(*CachedJS); ok {
			if cachedJS.Timestamp.Before(cutoff) {
				cu.cache.Delete(key)
				logger.Debug("Cleared cached compilation for %s (age: %v)", key, time.Since(cachedJS.Timestamp))
			}
		}
		return true
	})
}
