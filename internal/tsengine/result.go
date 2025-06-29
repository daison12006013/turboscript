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

// TSResult represents the result of executing a TypeScript query function.
//
// This structure contains the database operation details returned by
// TypeScript code, including the SQL query, parameters, and execution flags.
type TSResult struct {
	Operation        string `json:"operation"`
	Table            string `json:"table"`
	RawQuery         string `json:"query"`
	Params           []any  `json:"params"`
	SingleRecord     bool   `json:"single_record"`
	ExplicitNotFound bool   `json:"explicit_not_found"`
	NoQueryFunction  bool   `json:"__no_query_function"`
}
