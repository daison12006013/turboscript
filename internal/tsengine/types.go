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

// Package tsengine provides JavaScript runtime management utilities.
package tsengine

import (
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// Common string constants.
const (
	NullString      = "null"
	UndefinedString = "undefined"
	JSONFormat      = "json"
)

// CachedJS holds compiled JavaScript with metadata.
type CachedJS struct {
	Code      string
	Timestamp time.Time
	FileSize  int64
	ModTime   time.Time
}

// DataSource represents a data source configuration from TypeScript.
type DataSource struct {
	Name             string            `json:"name"`
	Source           string            `json:"source"`
	SingleRecord     bool              `json:"single_record,omitempty"`
	ExplicitNotFound bool              `json:"explicit_not_found,omitempty"`
	Query            string            `json:"query,omitempty"`    // For database source
	Params           []any             `json:"params,omitempty"`   // For database source
	URL              string            `json:"url,omitempty"`      // For URL source
	Method           string            `json:"method,omitempty"`   // For URL source (default: GET)
	Headers          map[string]string `json:"headers,omitempty"`  // For URL source
	Body             any               `json:"body,omitempty"`     // For URL source
	Timeout          int               `json:"timeout,omitempty"`  // For URL source (in milliseconds)
	Key              string            `json:"key,omitempty"`      // For Redis source
	Pattern          string            `json:"pattern,omitempty"`  // For Redis source
	Command          string            `json:"command,omitempty"`  // For Redis source
	Path             string            `json:"path,omitempty"`     // For file source
	Format           string            `json:"format,omitempty"`   // For file source
	Encoding         string            `json:"encoding,omitempty"` // For file source
	Endpoint         string            `json:"endpoint,omitempty"` // For API source
	Auth             map[string]any    `json:"auth,omitempty"`     // For API source
}

// JSRuntime wraps goja runtime and its registry.
type JSRuntime struct {
	Runtime  *goja.Runtime
	Registry *require.Registry
}
