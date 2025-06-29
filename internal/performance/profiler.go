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

// Package performance provides profiling server capabilities for TurboScript applications.
//
// This file implements a dedicated pprof server that runs on a separate port,
// providing access to Go's built-in profiling tools without interfering with
// the main application server.
package performance

import (
	"net/http"
	// #nosec G108: pprof endpoint is intentionally exposed for debugging/profiling
	_ "net/http/pprof" // Import pprof for profiling endpoints
	"sync"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
)

var (
	profileServerStarted sync.Once // Ensures profiling server starts only once
)

// StartProfileServer starts the pprof profiling server on a separate port.
//
// This function initializes a dedicated HTTP server for Go's pprof profiling tools.
// The server provides various profiling endpoints for performance analysis:
//
// Available Endpoints:
//   - http://localhost:6060/debug/pprof/ - Main profiling page with overview
//   - http://localhost:6060/debug/pprof/profile - CPU profile (30 seconds)
//   - http://localhost:6060/debug/pprof/heap - Memory heap profile
//   - http://localhost:6060/debug/pprof/goroutine - Goroutine dump
//   - http://localhost:6060/debug/pprof/trace?seconds=5 - Execution trace
//
// The server runs in a separate goroutine and uses sync.Once to ensure
// it's started only once, even if called multiple times.
//
// Parameters:
//   - port: Port number for the profiling server (defaults to "6060" if empty)
func StartProfileServer(port string) {
	profileServerStarted.Do(func() {
		if port == "" {
			port = "6060"
		}

		logger.Info("[PPROF] Starting profiling server on port %s", port)
		logger.Info("[PPROF] Access profiling at: http://localhost:%s/debug/pprof/", port)
		logger.Info("[PPROF] CPU Profile: http://localhost:%s/debug/pprof/profile", port)
		logger.Info("[PPROF] Memory Profile: http://localhost:%s/debug/pprof/heap", port)
		logger.Info("[PPROF] Goroutine Profile: http://localhost:%s/debug/pprof/goroutine", port)

		go func() {
			server := &http.Server{
				Addr:         ":" + port,
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 10 * time.Second,
			}

			if err := server.ListenAndServe(); err != nil {
				logger.Error("[PPROF] Failed to start profiling server: %v", err)
			}
		}()
	})
}

// LogProfilingInstructions logs detailed instructions for using the profiling server.
//
// This function outputs comprehensive instructions for collecting and analyzing
// various types of performance profiles. It provides ready-to-use curl commands
// and analysis tools for developers to diagnose performance issues.
//
// The instructions cover:
//   - CPU profiling for identifying performance bottlenecks
//   - Memory profiling for detecting memory leaks
//   - Goroutine profiling for concurrency analysis
//   - Execution tracing for detailed runtime behavior
//   - Analysis tools and web interfaces for profile examination
func LogProfilingInstructions() {
	LogProfilingInstructionsWithPort("6060")
}

// LogProfilingInstructionsWithPort logs pprof usage instructions with a specific port.
func LogProfilingInstructionsWithPort(port string) {
	logger.Info("[PPROF] === Profiling Instructions ===")
	logger.Info("[PPROF] 1. CPU Profile (30 seconds): curl http://localhost:%s/debug/pprof/profile > cpu.prof", port)
	logger.Info("[PPROF] 2. Memory Profile: curl http://localhost:%s/debug/pprof/heap > mem.prof", port)
	logger.Info("[PPROF] 3. Goroutine Profile: curl http://localhost:%s/debug/pprof/goroutine > goroutine.prof", port)
	logger.Info("[PPROF] 4. Analyze with: go tool pprof cpu.prof")
	logger.Info("[PPROF] 5. Web UI: go tool pprof -http=:8081 cpu.prof")
	logger.Info("[PPROF] 6. Trace (5 seconds): curl http://localhost:%s/debug/pprof/trace?seconds=5 > trace.out", port)
	logger.Info("[PPROF] 7. View trace: go tool trace trace.out")
	logger.Info("[PPROF] =============================")
}
