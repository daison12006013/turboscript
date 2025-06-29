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

// Package tsengine provides response transformation utilities for TypeScript functions.
package tsengine

import (
	"database/sql"
)

// ResponseUtils handles TypeScript handle function execution and response processing.
type ResponseUtils struct {
	cacheUtils         *CacheUtils
	runtimeUtils       *RuntimeUtils
	errorUtils         *ErrorUtils
	turboMarkdownUtils *TurboMarkdownUtils
	db                 *sql.DB // Database connection for runQuery functionality (legacy)
	dbManager          any     // Database manager for multiple connections
}

// NewResponseUtils creates a new response utilities instance.
func NewResponseUtils(cacheUtils *CacheUtils, runtimeUtils *RuntimeUtils, errorUtils *ErrorUtils) *ResponseUtils {
	return &ResponseUtils{
		cacheUtils:         cacheUtils,
		runtimeUtils:       runtimeUtils,
		errorUtils:         errorUtils,
		turboMarkdownUtils: NewTurboMarkdownUtils(""), // Initialize with empty base path
		db:                 nil,                       // Will be set via SetDatabase (legacy)
		dbManager:          nil,                       // Will be set via SetDatabaseManager
	}
}

// SetDatabase sets the database connection for runQuery functionality.
func (ru *ResponseUtils) SetDatabase(db *sql.DB) {
	ru.db = db
}

// SetDatabaseManager sets the database manager for multi-connection support.
func (ru *ResponseUtils) SetDatabaseManager(dbManager any) {
	ru.dbManager = dbManager
}

// SetMarkdownBasePath sets the base path for turboMarkdownHtml function.
func (ru *ResponseUtils) SetMarkdownBasePath(basePath string) {
	ru.turboMarkdownUtils.SetBasePath(basePath)
}
