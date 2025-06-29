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

// Package tsengine provides shared utilities for database query handling.
package tsengine

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/dop251/goja"
)

// TurboQueryUtils provides shared turboQuery functionality.
type TurboQueryUtils struct {
}

// NewTurboQueryUtils creates a new turboQuery utilities instance.
func NewTurboQueryUtils() *TurboQueryUtils {
	return &TurboQueryUtils{}
}

// ExecuteQuery executes a database query and returns the results as a Goja value.
//
// This method provides shared database query execution functionality for both sync
// and async turboQuery implementations. It handles parameter parsing, query execution,
// result processing, and error handling in a consistent manner.
//
// Supports two call formats:
//  1. Default: turboQuery(query, params) - uses default connection
//  2. Object: turboQuery({query, bindings?, connection?}) - supports connection switching
//
// Parameters:
//   - call: The Goja function call containing query and parameters
//   - rt: The JavaScript runtime for value conversion and error handling
//   - db: Default database connection to use for legacy calls or when no manager is available
//   - dbManager: Database manager for multi-connection support (optional)
//   - debugPrefix: Prefix for debug logging to distinguish call sources
//
// Returns a Goja value containing the query results (array of objects) or throws
// a JavaScript error if the query fails or database is unavailable.
func (tqu *TurboQueryUtils) ExecuteQuery(call goja.FunctionCall, rt *goja.Runtime, db *sql.DB, dbManager any, debugPrefix string) goja.Value {
	// Parse turboQuery arguments
	query, params, connectionName, err := tqu.ParseTurboQueryArgs(call, rt)
	if err != nil {
		panic(rt.NewGoError(err))
	}

	targetDB := tqu.getTargetDatabase(connectionName, dbManager, db, rt)
	if targetDB == nil {
		panic(rt.NewGoError(fmt.Errorf("database connection not available")))
	}

	logger.Debug("%s called with query: %s, params: %+v, connection: %s", debugPrefix, query, params, connectionName)

	// Execute the query
	rows, err := targetDB.Query(query, params...)
	if err != nil {
		panic(rt.NewGoError(fmt.Errorf("query execution failed: %w", err)))
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			logger.Error("Failed to close database rows: %v", closeErr)
		}
	}()

	// Convert rows to map slice
	results := make([]map[string]any, 0)
	columns, err := rows.Columns()
	if err != nil {
		panic(rt.NewGoError(fmt.Errorf("failed to get columns: %w", err)))
	}

	for rows.Next() {
		values := make([]any, len(columns))
		valuePtrs := make([]any, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			panic(rt.NewGoError(fmt.Errorf("failed to scan row: %w", err)))
		}

		rowMap := make(map[string]any)
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				val = string(b)
			}
			rowMap[col] = val
		}
		results = append(results, rowMap)
	}

	if err := rows.Err(); err != nil {
		panic(rt.NewGoError(fmt.Errorf("rows iteration error: %w", err)))
	}

	// Return the results directly
	return rt.ToValue(results)
}

// getTargetDatabase gets the appropriate database connection.
func (tqu *TurboQueryUtils) getTargetDatabase(connectionName string, dbManager any, db *sql.DB, rt *goja.Runtime) *sql.DB {
	if connectionName == "" || dbManager == nil {
		return db
	}

	// Try to get connection from database manager using reflection to avoid circular imports
	dbMgr, ok := dbManager.(interface {
		GetConnection(string) (*sql.DB, error)
	})
	if !ok {
		panic(rt.NewGoError(fmt.Errorf("database manager does not support GetConnection method")))
	}

	targetDB, err := dbMgr.GetConnection(connectionName)
	if err != nil {
		panic(rt.NewGoError(fmt.Errorf("failed to get database connection '%s': %w", connectionName, err)))
	}

	return targetDB
}

// ParseTurboQueryArgs parses turboQuery arguments and returns query, params, and connection name.
// Supports both legacy format turboQuery(query, params) and object format turboQuery({query, bindings?, connection?}).
func (tqu *TurboQueryUtils) ParseTurboQueryArgs(call goja.FunctionCall, _ *goja.Runtime) (string, []any, string, error) {
	if len(call.Arguments) == 0 {
		return "", nil, "", fmt.Errorf("turboQuery requires at least 1 argument")
	}

	firstArg := call.Argument(0)

	// Check if first argument is an object (new format)
	if firstArg.ExportType().Kind() == reflect.TypeOf(map[string]any{}).Kind() {
		// Object format: {query, bindings?, connection?}
		optionsValue := firstArg.Export()
		if options, ok := optionsValue.(map[string]any); ok {
			return tqu.parseObjectFormat(options)
		}
		return "", nil, "", fmt.Errorf("first argument must be a string or options object")
	}

	// Legacy format: turboQuery(query, params)
	return tqu.parseLegacyFormat(call)
}

// parseObjectFormat parses turboQuery arguments in object format.
func (tqu *TurboQueryUtils) parseObjectFormat(options map[string]any) (string, []any, string, error) {
	var query string
	var params []any
	var connectionName string

	// Extract query
	if queryVal, exists := options["query"]; exists {
		if queryStr, ok := queryVal.(string); ok {
			query = queryStr
		} else {
			return "", nil, "", fmt.Errorf("query must be a string")
		}
	} else {
		return "", nil, "", fmt.Errorf("query field is required in options object")
	}

	// Extract bindings (optional)
	if bindingsVal, exists := options["bindings"]; exists && bindingsVal != nil {
		if paramsSlice, ok := bindingsVal.([]any); ok {
			params = paramsSlice
		} else {
			return "", nil, "", fmt.Errorf("bindings must be an array")
		}
	}

	// Extract connection (optional)
	if connVal, exists := options["connection"]; exists && connVal != nil {
		if connStr, ok := connVal.(string); ok {
			connectionName = connStr
		} else {
			return "", nil, "", fmt.Errorf("connection must be a string")
		}
	}

	return query, params, connectionName, nil
}

// parseLegacyFormat parses turboQuery arguments in legacy format.
func (tqu *TurboQueryUtils) parseLegacyFormat(call goja.FunctionCall) (string, []any, string, error) {
	// Legacy format: turboQuery(query, params)
	query := call.Argument(0).String()
	var params []any

	if len(call.Arguments) > 1 && !goja.IsNull(call.Arguments[1]) && !goja.IsUndefined(call.Arguments[1]) {
		paramsArg := call.Arguments[1].Export()
		if paramsSlice, ok := paramsArg.([]any); ok {
			params = paramsSlice
		}
	}

	return query, params, "", nil // Empty connection name for legacy format
}
