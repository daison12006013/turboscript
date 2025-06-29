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

// Package tsengine provides turboJob utilities for background job processing.
package tsengine

import (
	"fmt"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/dop251/goja"
)

// JobManager represents a job manager interface to avoid import cycles.
type JobManager interface {
	DispatchJob(jobPath string, payload map[string]any) error
}

// DBJobManager extends the basic JobManager interface with database-specific methods.
// To avoid import cycles, we use a type assertion in ExecuteJob when a database
// job manager is detected.
type DBJobManager interface {
	JobManager
	DispatchJobWithReturn(jobPath string, payload map[string]any) (string, error)
}

// TurboJobUtils provides shared turboJob functionality.
type TurboJobUtils struct {
	jobManager JobManager
}

// NewTurboJobUtils creates a new turboJob utilities instance.
func NewTurboJobUtils(jobManager JobManager) *TurboJobUtils {
	return &TurboJobUtils{
		jobManager: jobManager,
	}
}

// ExecuteJob dispatches a background job for processing.
// This function is used by both sync and async turboJob implementations.
// It handles parameter parsing, job validation, and job dispatching.
//
// Parameters:
//   - call: The goja function call containing job path and payload
//   - rt: The goja runtime for error reporting
//
// Returns:
//   - goja.Value: A promise that resolves when the job is queued
func (tju *TurboJobUtils) ExecuteJob(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	logger.Debug("🔄 turboJob called with %d arguments", len(call.Arguments))

	// Validate argument count
	if len(call.Arguments) < 2 {
		panic(rt.NewGoError(fmt.Errorf("turboJob requires 2 arguments: jobPath and payload")))
	}

	// Extract job path
	jobPath := call.Arguments[0].String()
	if jobPath == "" {
		panic(rt.NewGoError(fmt.Errorf("job path cannot be empty")))
	}

	// Extract payload
	payloadValue := call.Arguments[1]
	var payload map[string]any

	if goja.IsUndefined(payloadValue) || goja.IsNull(payloadValue) {
		payload = make(map[string]any)
	} else {
		// Convert goja value to Go map
		exported := payloadValue.Export()
		if payloadMap, ok := exported.(map[string]any); ok {
			payload = payloadMap
		} else {
			panic(rt.NewGoError(fmt.Errorf("payload must be an object")))
		}
	}

	logger.Debug("📝 Dispatching job: path=%s, payload keys=%v", jobPath, getMapKeys(payload))

	// Check if job manager is available
	if tju.jobManager == nil {
		panic(rt.NewGoError(fmt.Errorf("job manager is not initialized")))
	}

	// Try to use the enhanced DBJobManager interface if available
	if dbJobManager, ok := tju.jobManager.(DBJobManager); ok {
		// Dispatch the job with UID return
		jobUID, err := dbJobManager.DispatchJobWithReturn(jobPath, payload)
		if err != nil {
			panic(rt.NewGoError(fmt.Errorf("failed to dispatch job: %w", err)))
		}

		logger.Debug("✅ Job dispatched successfully: %s (UID: %s)", jobPath, jobUID)

		// Return the job UID
		return rt.ToValue(jobUID)
	}

	// Fall back to the basic interface
	if err := tju.jobManager.DispatchJob(jobPath, payload); err != nil {
		panic(rt.NewGoError(fmt.Errorf("failed to dispatch job: %w", err)))
	}

	logger.Debug("✅ Job dispatched successfully: %s", jobPath)

	// Return empty string as job ID for backward compatibility
	return rt.ToValue("")
}

// getMapKeys returns the keys of a map for logging purposes.
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
