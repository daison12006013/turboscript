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

// Package tsengine provides TypeScript execution capabilities for TurboScript.
//
// This package implements a TypeScript execution engine using the goja JavaScript VM
// and esbuild for compilation. It provides a secure, sandboxed environment for
// executing TypeScript code with built-in utilities and performance optimizations.
package tsengine

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/email"
)

// TSExecutor provides TypeScript code execution capabilities with modular architecture.
// It separates concerns into specialized utilities for better maintainability.
type TSExecutor struct {
	// Core utilities
	compiler           *CompilerUtils
	runtimeUtils       *RuntimeUtils
	cacheUtils         *CacheUtils
	errorUtils         *ErrorUtils
	responseUtils      *ResponseUtils
	asyncResponseUtils *AsyncResponseUtils // Async capabilities
	preserveResponse   bool                // Preserve JSON field order (from config)
	fileResolver       *FileResolver       // File resolution utility
}

// NewIsolatedTSExecutor creates a completely isolated TypeScript executor instance
// specifically for background jobs and email processing to prevent interference
// with HTTP request processing.
func NewIsolatedTSExecutor(preserveResponse bool) *TSExecutor {
	// Create completely new utility instances (no sharing)
	compiler := NewCompilerUtils()
	runtimeUtils := NewRuntimeUtils()
	cacheUtils := NewCacheUtils(compiler)
	errorUtils := NewErrorUtils()
	responseUtils := NewResponseUtils(cacheUtils, runtimeUtils, errorUtils)
	asyncResponseUtils := NewAsyncResponseUtils(cacheUtils, runtimeUtils, errorUtils)
	fileResolver := GetRecommendedResolver()

	executor := &TSExecutor{
		compiler:           compiler,
		runtimeUtils:       runtimeUtils,
		cacheUtils:         cacheUtils,
		errorUtils:         errorUtils,
		responseUtils:      responseUtils,
		asyncResponseUtils: asyncResponseUtils,
		preserveResponse:   preserveResponse,
		fileResolver:       fileResolver,
	}

	// Start isolated async mode with its own event loop
	executor.asyncResponseUtils.StartEventLoop()

	return executor
}

// NewIsolatedTSExecutorWithResolver creates an isolated TypeScript executor with a custom file resolver.
func NewIsolatedTSExecutorWithResolver(preserveResponse bool, fileResolver *FileResolver) *TSExecutor {
	// Create completely new utility instances (no sharing)
	compiler := NewCompilerUtils()
	runtimeUtils := NewRuntimeUtils()
	cacheUtils := NewCacheUtils(compiler)
	errorUtils := NewErrorUtils()
	responseUtils := NewResponseUtils(cacheUtils, runtimeUtils, errorUtils)
	asyncResponseUtils := NewAsyncResponseUtils(cacheUtils, runtimeUtils, errorUtils)

	executor := &TSExecutor{
		compiler:           compiler,
		runtimeUtils:       runtimeUtils,
		cacheUtils:         cacheUtils,
		errorUtils:         errorUtils,
		responseUtils:      responseUtils,
		asyncResponseUtils: asyncResponseUtils,
		preserveResponse:   preserveResponse,
		fileResolver:       fileResolver,
	}

	// Start isolated async mode with its own event loop
	executor.asyncResponseUtils.StartEventLoop()

	return executor
}

// NewIsolatedTSExecutorWithResolverAndConfig creates an isolated TypeScript executor with a custom file resolver and TypeScript configuration.
func NewIsolatedTSExecutorWithResolverAndConfig(preserveResponse bool, fileResolver *FileResolver, tsConfig *config.TypeScriptCompilerConfig) *TSExecutor {
	// Create completely new utility instances (no sharing)
	compiler := NewCompilerUtilsWithConfig(tsConfig)
	runtimeUtils := NewRuntimeUtils()
	cacheUtils := NewCacheUtils(compiler)
	errorUtils := NewErrorUtils()
	responseUtils := NewResponseUtils(cacheUtils, runtimeUtils, errorUtils)
	asyncResponseUtils := NewAsyncResponseUtils(cacheUtils, runtimeUtils, errorUtils)

	executor := &TSExecutor{
		compiler:           compiler,
		runtimeUtils:       runtimeUtils,
		cacheUtils:         cacheUtils,
		errorUtils:         errorUtils,
		responseUtils:      responseUtils,
		asyncResponseUtils: asyncResponseUtils,
		preserveResponse:   preserveResponse,
		fileResolver:       fileResolver,
	}

	// Start isolated async mode with its own event loop
	executor.asyncResponseUtils.StartEventLoop()

	return executor
}

// SetDatabase sets the database connection for both sync and async execution.
func (e *TSExecutor) SetDatabase(db any) {
	// Type assert to sql.DB
	sqlDB, ok := db.(*sql.DB)
	if !ok {
		return // Skip if not a valid database connection
	}

	// Set for response utilities (sync execution)
	e.responseUtils.SetDatabase(sqlDB)

	// Set for async response utilities if async is enabled
	if e.asyncResponseUtils != nil {
		e.asyncResponseUtils.SetDatabase(sqlDB)
	}
}

// SetDatabaseManager sets the database manager for both sync and async execution.
func (e *TSExecutor) SetDatabaseManager(dbManager any) {
	// Type assert to DatabaseManager (defined in config package)
	// Using any to avoid circular import
	if dbManager == nil {
		return
	}

	// Set for response utilities (sync execution)
	e.responseUtils.SetDatabaseManager(dbManager)

	// Set for async response utilities if async is enabled
	if e.asyncResponseUtils != nil {
		e.asyncResponseUtils.SetDatabaseManager(dbManager)
	}
}

// SetServer sets the server reference for broadcasting capabilities.
func (e *TSExecutor) SetServer(server interface{}) {
	// Set for async response utilities if async is enabled
	if e.asyncResponseUtils != nil {
		e.asyncResponseUtils.SetServer(server)
	}
}

// TerminateAsync terminates the async event loop and cleans up resources.
func (e *TSExecutor) TerminateAsync() {
	if e.asyncResponseUtils != nil {
		e.asyncResponseUtils.TerminateEventLoop()
	}
}

// ResolveFile resolves a TypeScript/JavaScript file path using the configured file resolver.
func (e *TSExecutor) ResolveFile(filePath string) (string, error) {
	return e.fileResolver.ResolveFile(filePath)
}

// ResolveFileQuiet resolves a file path quietly (minimal logging).
func (e *TSExecutor) ResolveFileQuiet(filePath string) (string, error) {
	return e.fileResolver.ResolveFileQuiet(filePath)
}

// ExecuteHandleAutoWithTimeout automatically chooses sync or async execution with configurable timeout.
func (e *TSExecutor) ExecuteHandleAutoWithTimeout(tsPath string, event map[string]any, timeoutSeconds int) (json.RawMessage, error) {
	// Resolve the actual file path first
	resolvedPath, err := e.fileResolver.ResolveFileQuiet(tsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve file %s: %w", tsPath, err)
	}

	// Always use async execution
	return e.asyncResponseUtils.ExecuteHandleAsyncWithTimeout(resolvedPath, event, e.preserveResponse, timeoutSeconds)
}

// ExecuteHandleAsyncWithTimeout calls the handle function with async support and configurable timeout.
func (e *TSExecutor) ExecuteHandleAsyncWithTimeout(tsPath string, event map[string]any, timeoutSeconds int) (json.RawMessage, error) {
	return e.asyncResponseUtils.ExecuteHandleAsyncWithTimeout(tsPath, event, e.preserveResponse, timeoutSeconds)
}

// SetJobManager sets the job manager for background job processing.
func (e *TSExecutor) SetJobManager(jobManager JobManager) {
	if e.asyncResponseUtils != nil && e.asyncResponseUtils.eventLoop != nil {
		turboJobUtils := NewTurboJobUtils(jobManager)
		e.asyncResponseUtils.eventLoop.SetTurboJobUtils(turboJobUtils)
	}
}

// SetEmailService sets the email service for email sending.
func (e *TSExecutor) SetEmailService(emailService *email.Service) {
	if e.asyncResponseUtils != nil && e.asyncResponseUtils.eventLoop != nil {
		turboEmailUtils := NewTurboEmailUtils(emailService)
		e.asyncResponseUtils.eventLoop.SetTurboEmailUtils(turboEmailUtils)
	}
}

// SetMarkdownBasePath sets the base path for markdown and HTML file resolution.
func (e *TSExecutor) SetMarkdownBasePath(basePath string) {
	if e.responseUtils != nil {
		e.responseUtils.SetMarkdownBasePath(basePath)
	}
	if e.asyncResponseUtils != nil {
		e.asyncResponseUtils.SetMarkdownBasePath(basePath)
	}
}

// SetCacheConfig sets the cache configuration for turboCache operations.
func (e *TSExecutor) SetCacheConfig(cacheConfig *config.CacheConfig) {
	if e.asyncResponseUtils != nil {
		e.asyncResponseUtils.SetCacheConfig(cacheConfig)
	}
}
