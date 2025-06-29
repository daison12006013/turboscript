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

// Package performance provides goroutine debugging and leak detection capabilities
// for the TurboScript web framework.
//
// This package implements tools to monitor and debug goroutine behavior during
// request processing. The primary component is GoroutineDebugger, which helps
// identify potential goroutine leaks by comparing goroutine states before and
// after request processing.
//
// The debugger is designed to filter out expected background goroutines (such as
// FastHTTP worker pools and database connection managers) to focus on actual
// application-level leaks that may indicate bugs in user code.
//
// Example usage:
//
//	debugger := NewGoroutineDebugger()
//	// ... process request ...
//	hasLeaks := debugger.CheckForLeaks("request-123")
//	if hasLeaks {
//		log.Warn("Potential goroutine leak detected")
//	}
package performance

import (
	"runtime"
	"sort"
	"strings"

	"github.com/daison12006013/turboscript/internal/logger"
)

// GoroutineDebugger helps track and debug goroutine leaks during request processing.
//
// The debugger captures initial goroutine stack traces and can later compare
// them with final stack traces to identify new goroutines that may indicate leaks.
// It filters out expected background goroutines to focus on actual leaks.
type GoroutineDebugger struct {
	// initialCount stores the number of goroutines present when the debugger was created.
	// This serves as the baseline for detecting new goroutines during leak analysis.
	initialCount int

	// initialStack maps function names to their goroutine counts at debugger creation time.
	// The key is the function name from the stack trace, and the value is how many
	// goroutines were created by that function initially.
	initialStack map[string]int
}

// NewGoroutineDebugger creates a new goroutine debugger instance.
//
// This function captures the current state of goroutines as a baseline
// for later comparison. It records both the count and stack traces
// of existing goroutines.
//
// Returns a configured GoroutineDebugger ready for leak detection.
func NewGoroutineDebugger() *GoroutineDebugger {
	gd := &GoroutineDebugger{
		initialCount: runtime.NumGoroutine(),
		initialStack: make(map[string]int),
	}

	gd.captureStack()
	return gd
}

// captureStack captures the current goroutine stack traces for baseline analysis.
//
// This method takes a comprehensive snapshot of all current goroutine stack traces,
// parsing them to identify the functions that created each goroutine. The parsing
// process extracts meaningful function names from the stack traces and stores them
// in a map with occurrence counts.
//
// The captured data serves as a baseline for later comparison during leak detection.
// Stack traces are obtained using runtime.Stack with the all-goroutines flag set to true.
//
// This method is called automatically during debugger initialization and should not
// be called directly by users of this package.
func (gd *GoroutineDebugger) captureStack() {
	buf := make([]byte, 1024*1024)
	n := runtime.Stack(buf, true)
	stackTrace := string(buf[:n])

	// Parse goroutine stacks
	stacks := strings.Split(stackTrace, "\n\n")
	for _, stack := range stacks {
		if strings.TrimSpace(stack) == "" {
			continue
		}

		lines := strings.Split(stack, "\n")
		if len(lines) > 0 {
			gd.processGoroutineStack(lines)
		}
	}
}

func (gd *GoroutineDebugger) processGoroutineStack(lines []string) {
	// Get the first line which contains goroutine info
	firstLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(firstLine, "goroutine") {
		return
	}

	// Get the function that created this goroutine
	creatorFunc := gd.extractCreatorFunc(lines)
	if creatorFunc != "" {
		gd.initialStack[creatorFunc]++
	}
}

func (gd *GoroutineDebugger) extractCreatorFunc(lines []string) string {
	for i, line := range lines {
		if i > 0 && strings.Contains(line, "(") && !strings.Contains(line, "+0x") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// CheckForLeaks analyzes current goroutines for potential leaks compared to the initial state.
//
// This method compares the current goroutine count and stack traces with the initial
// baseline captured during debugger creation. It identifies new goroutines and filters
// out known background goroutines (such as FastHTTP worker pools) to focus on actual
// potential leaks.
//
// The requestID parameter is used for logging correlation to track which request
// may have caused the goroutine leak.
//
// Returns true if actual goroutine leaks are detected (excluding expected background
// goroutines), false otherwise. All findings are logged with appropriate severity levels.
func (gd *GoroutineDebugger) CheckForLeaks(requestID string) bool {
	currentCount := runtime.NumGoroutine()
	diff := currentCount - gd.initialCount

	if diff <= 0 {
		return false
	}

	logger.Warn("[GOROUTINE] [%s] Detected %d new goroutines", requestID, diff)

	currentStack := gd.getCurrentGoroutineStack()
	newGoroutines, ignoredGoroutines := gd.categorizeGoroutines(currentStack, requestID)

	return gd.logAndReportLeaks(requestID, newGoroutines, ignoredGoroutines)
}

// getCurrentGoroutineStack parses the current goroutine stack traces into a map.
func (gd *GoroutineDebugger) getCurrentGoroutineStack() map[string]int {
	currentStack := make(map[string]int)
	buf := make([]byte, 1024*1024)
	n := runtime.Stack(buf, true)
	stackTrace := string(buf[:n])

	stacks := strings.Split(stackTrace, "\n\n")
	for _, stack := range stacks {
		if creatorFunc := gd.extractCreatorFunction(stack); creatorFunc != "" {
			currentStack[creatorFunc]++
		}
	}
	return currentStack
}

// extractCreatorFunction extracts the creator function from a goroutine stack trace.
func (gd *GoroutineDebugger) extractCreatorFunction(stack string) string {
	if strings.TrimSpace(stack) == "" {
		return ""
	}

	lines := strings.Split(stack, "\n")
	if len(lines) == 0 {
		return ""
	}

	firstLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(firstLine, "goroutine") {
		return ""
	}

	for i, line := range lines {
		if i > 0 && strings.Contains(line, "(") && !strings.Contains(line, "+0x") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// categorizeGoroutines separates new goroutines into leaked vs expected background goroutines.
func (gd *GoroutineDebugger) categorizeGoroutines(currentStack map[string]int, requestID string) (map[string]int, map[string]int) {
	logger.Info("[GOROUTINE] [%s] Analyzing goroutine sources:", requestID)
	newGoroutines := make(map[string]int)
	ignoredGoroutines := make(map[string]int)

	for funcName, count := range currentStack {
		initialCount := gd.initialStack[funcName]
		if count > initialCount {
			diff := count - initialCount
			if gd.isKnownBackgroundGoroutine(funcName) {
				ignoredGoroutines[funcName] = diff
			} else {
				newGoroutines[funcName] = diff
			}
		}
	}
	return newGoroutines, ignoredGoroutines
}

// logAndReportLeaks logs the categorized goroutine information and returns whether actual leaks were found.
func (gd *GoroutineDebugger) logAndReportLeaks(requestID string, newGoroutines, ignoredGoroutines map[string]int) bool {
	// Log ignored goroutines for reference
	if len(ignoredGoroutines) > 0 {
		logger.Debug("[GOROUTINE] [%s] Ignored known background goroutines:", requestID)
		for fn, count := range ignoredGoroutines {
			logger.Debug("[GOROUTINE] [%s] ~%d goroutines from: %s (expected)", requestID, count, fn)
		}
	}

	// Sort and log actual leaks
	hasActualLeaks := len(newGoroutines) > 0
	if hasActualLeaks {
		gd.logSortedGoroutines(requestID, newGoroutines)
	} else {
		logger.Debug("[GOROUTINE] [%s] All new goroutines are expected background operations", requestID)
	}

	if len(newGoroutines) == 0 && len(ignoredGoroutines) == 0 {
		logger.Warn("[GOROUTINE] [%s] Could not identify source of new goroutines", requestID)
	}

	return hasActualLeaks
}

// logSortedGoroutines logs new goroutines sorted by count for better readability.
func (gd *GoroutineDebugger) logSortedGoroutines(requestID string, newGoroutines map[string]int) {
	type funcCount struct {
		function string
		count    int
	}

	sorted := make([]funcCount, 0, len(newGoroutines))
	for fn, count := range newGoroutines {
		sorted = append(sorted, funcCount{fn, count})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	for _, fc := range sorted {
		logger.Warn("[GOROUTINE] [%s] +%d goroutines from: %s", requestID, fc.count, fc.function)
	}
}

// isKnownBackgroundGoroutine determines if a function represents an expected background goroutine.
//
// This method checks if the provided function name matches patterns of known background
// goroutines that are expected to be created during normal application operation.
// These include FastHTTP worker pools, HTTP server operations, database connection
// management, and the application's own performance monitoring goroutines.
//
// Background goroutines are filtered out during leak detection to avoid false positives
// and focus on actual goroutine leaks that may indicate problems in the application code.
//
// The funcName parameter should be the function name extracted from a goroutine's
// stack trace.
//
// Returns true if the function is recognized as a known background operation,
// false if it represents a potentially leaked goroutine that should be investigated.
func (gd *GoroutineDebugger) isKnownBackgroundGoroutine(funcName string) bool {
	knownBackgroundGoroutines := []string{
		// FastHTTP background operations
		"github.com/valyala/fasthttp.(*workerPool).Start.func2",
		"github.com/valyala/fasthttp.updateServerDate.func1",
		"github.com/valyala/fasthttp.(*workerPool).workerFunc",
		"github.com/valyala/fasthttp.(*workerPool).getCh.func1",

		// HTTP server operations
		"net/http.(*Server).Serve",
		"net/http.(*Server).ListenAndServe",
		"net/http.(*conn).serve",

		// Database connection management
		"database/sql.(*DB).connectionOpener",

		// Performance monitoring (our own)
		"github.com/daison12006013/turboscript/internal/performance.StartPeriodicMetrics.func1.1",
		"github.com/daison12006013/turboscript/internal/performance.StartProfileServer.func1.1",

		// Generic patterns for HTTP/network operations
		"internal/poll.runtime_pollWait",
		"net.(*netFD).accept",
		"net.(*TCPListener).accept",
		"bufio.(*Reader).fill",
		"bufio.(*Reader).Peek",
		"time.Sleep",
		"github.com/daison12006013/turboscript/internal/performance.",
	}

	for _, known := range knownBackgroundGoroutines {
		if strings.Contains(funcName, known) {
			return true
		}
	}

	return false
}
