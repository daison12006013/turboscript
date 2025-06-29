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

// Package performance provides real-time performance monitoring for TurboScript applications.
//
// This file implements the core monitoring system that tracks request metrics, response times,
// and component-specific performance data. It provides thread-safe metric collection and
// analysis capabilities for production deployments.
package performance

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
)

// Metrics holds comprehensive performance metrics for the application.
//
// This structure contains aggregate performance data collected during runtime,
// including request counts, timing information, and component-specific metrics.
type Metrics struct {
	RequestCount        int64   // Total number of requests processed
	TotalResponseTime   int64   // Cumulative response time in milliseconds
	AverageResponseTime float64 // Calculated average response time
	MaxResponseTime     int64   // Maximum response time observed
	MinResponseTime     int64   // Minimum response time observed
	TSExecutionTime     int64   // Total TypeScript execution time
	DBExecutionTime     int64   // Total database execution time
	ActiveRequests      int64   // Current number of active requests
}

// Performance monitor singleton - global metrics and synchronization.
var (
	globalMetrics = &Metrics{
		MinResponseTime: 9999999, // Start with high value for proper minimum calculation
	}
	requestIDCounter int64     // Atomic counter for generating unique request IDs
	metricsStarted   sync.Once // Ensures metrics initialization happens only once
)

// RequestContext holds detailed timing information for a single request.
//
// This structure tracks the lifecycle of an individual request through various
// processing stages, enabling detailed performance analysis and bottleneck identification.
type RequestContext struct {
	RequestID   string    // Unique identifier for the request
	StartTime   time.Time // Request start timestamp
	Method      string    // HTTP method (GET, POST, etc.)
	Path        string    // Request path
	TSFile      string    // TypeScript file handling the request
	TSStartTime time.Time // TypeScript execution start time
	TSEndTime   time.Time // TypeScript execution end time
	DBStartTime time.Time // Database operation start time
	DBEndTime   time.Time // Database operation end time

	// Granular timing fields for detailed analysis
	ParsingStartTime  time.Time          // Request parsing start time
	ParsingEndTime    time.Time          // Request parsing end time
	ResponseStartTime time.Time          // Response generation start time
	ResponseEndTime   time.Time          // Response generation end time
	ResponseTime      time.Duration      // Total response time
	StatusCode        int                // HTTP response status code
	MemoryBefore      uint64             // Memory usage before request processing
	MemoryAfter       uint64             // Memory usage after request processing
	GoroutinesBefore  int                // Goroutine count before request processing
	GoroutinesAfter   int                // Goroutine count after request processing
	goroutineDebugger *GoroutineDebugger // Goroutine debugging helper
}

// NewRequestContext creates a new request context with timing and resource tracking.
//
// This function initializes a new request context that will track the complete
// lifecycle of an HTTP request. It captures initial system state including
// memory usage and goroutine counts for resource leak detection.
//
// Parameters:
//   - method: HTTP method (GET, POST, etc.)
//   - path: Request path being processed
//
// Returns a configured RequestContext ready for performance tracking.
func NewRequestContext(method, path string) *RequestContext {
	requestID := fmt.Sprintf("req_%d", atomic.AddInt64(&requestIDCounter, 1))

	// Capture initial memory and goroutine stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	ctx := &RequestContext{
		RequestID:         requestID,
		StartTime:         time.Now(),
		Method:            method,
		Path:              path,
		MemoryBefore:      memStats.Alloc,
		GoroutinesBefore:  runtime.NumGoroutine(),
		goroutineDebugger: NewGoroutineDebugger(),
	}

	atomic.AddInt64(&globalMetrics.ActiveRequests, 1)

	logger.Info("[MONITORING] [%s] Request started: %s %s", requestID, method, path)
	logger.Debug("[MONITORING] [%s] Memory before: %d bytes, Goroutines: %d",
		requestID, ctx.MemoryBefore, ctx.GoroutinesBefore)

	return ctx
}

// StartRequestParsing marks the start of request parsing timing.
//
// This method should be called before parsing request parameters, headers,
// and body content. It helps identify parsing performance bottlenecks.
func (ctx *RequestContext) StartRequestParsing() {
	ctx.ParsingStartTime = time.Now()
	logger.Debug("[MONITORING] [%s] Request parsing started", ctx.RequestID)
}

// EndRequestParsing marks the end of request parsing and records timing.
//
// This method should be called after request parsing is complete.
// It logs performance warnings if parsing takes longer than 10ms,
// which may indicate complex request processing.
func (ctx *RequestContext) EndRequestParsing() {
	ctx.ParsingEndTime = time.Now()
	duration := ctx.ParsingEndTime.Sub(ctx.ParsingStartTime)

	logger.Debug("[MONITORING] [%s] Request parsing completed in %v", ctx.RequestID, duration)
	if duration > 10*time.Millisecond {
		logger.Warn("[MONITORING] [%s] SLOW request parsing: %v (>10ms)", ctx.RequestID, duration)
	}
}

// StartResponseProcessing marks the start of response processing timing.
//
// This method should be called before response transformation and serialization.
// It helps identify bottlenecks in response generation and formatting.
func (ctx *RequestContext) StartResponseProcessing() {
	ctx.ResponseStartTime = time.Now()
	logger.Debug("[MONITORING] [%s] Response processing started", ctx.RequestID)
}

// EndResponseProcessing marks the end of response processing and records timing.
//
// This method should be called after response processing is complete.
// It logs performance warnings if processing takes longer than 50ms,
// which may indicate complex response transformation logic.
func (ctx *RequestContext) EndResponseProcessing() {
	ctx.ResponseEndTime = time.Now()
	duration := ctx.ResponseEndTime.Sub(ctx.ResponseStartTime)

	logger.Debug("[MONITORING] [%s] Response processing completed in %v", ctx.RequestID, duration)
	if duration > 50*time.Millisecond {
		logger.Warn("[MONITORING] [%s] SLOW response processing: %v (>50ms)", ctx.RequestID, duration)
	}
}

// Finish completes the request timing and logs comprehensive performance metrics.
//
// This method should be called at the end of request processing to finalize
// all timing measurements and update global performance statistics. It captures
// final system state, calculates memory and goroutine differences, and logs
// detailed performance information.
//
// Parameters:
//   - statusCode: HTTP status code returned for the request
func (ctx *RequestContext) Finish(statusCode int) {
	ctx.StatusCode = statusCode
	ctx.ResponseTime = time.Since(ctx.StartTime)

	// Capture final memory and goroutine stats
	ctx.captureFinalStats()

	// Update global metrics
	ctx.updateGlobalMetrics()

	// Log performance information
	ctx.logPerformanceInfo(statusCode)
}

func (ctx *RequestContext) captureFinalStats() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	ctx.MemoryAfter = memStats.Alloc
	ctx.GoroutinesAfter = runtime.NumGoroutine()

	atomic.AddInt64(&globalMetrics.ActiveRequests, -1)
}

func (ctx *RequestContext) updateGlobalMetrics() {
	responseTimeMs := ctx.ResponseTime.Milliseconds()
	atomic.AddInt64(&globalMetrics.RequestCount, 1)
	atomic.AddInt64(&globalMetrics.TotalResponseTime, responseTimeMs)

	ctx.updateMinMaxResponseTimes(responseTimeMs)
	ctx.updateAverageResponseTime()
}

func (ctx *RequestContext) updateMinMaxResponseTimes(responseTimeMs int64) {
	// Update max response time
	for {
		currentMax := atomic.LoadInt64(&globalMetrics.MaxResponseTime)
		if responseTimeMs <= currentMax || atomic.CompareAndSwapInt64(&globalMetrics.MaxResponseTime, currentMax, responseTimeMs) {
			break
		}
	}

	// Update min response time
	for {
		currentMin := atomic.LoadInt64(&globalMetrics.MinResponseTime)
		if responseTimeMs >= currentMin || atomic.CompareAndSwapInt64(&globalMetrics.MinResponseTime, currentMin, responseTimeMs) {
			break
		}
	}
}

func (ctx *RequestContext) updateAverageResponseTime() {
	requestCount := atomic.LoadInt64(&globalMetrics.RequestCount)
	totalTime := atomic.LoadInt64(&globalMetrics.TotalResponseTime)
	if requestCount > 0 {
		globalMetrics.AverageResponseTime = float64(totalTime) / float64(requestCount)
	}
}

func (ctx *RequestContext) logPerformanceInfo(statusCode int) {
	memoryDiff := calculateMemoryDiff(ctx.MemoryBefore, ctx.MemoryAfter)
	goroutineDiff := ctx.GoroutinesAfter - ctx.GoroutinesBefore

	ctx.logBasicInfo(statusCode, memoryDiff, goroutineDiff)
	ctx.logComponentBreakdown()
	ctx.logPerformanceWarnings(memoryDiff, goroutineDiff)
}

func (ctx *RequestContext) logBasicInfo(statusCode int, memoryDiff int64, goroutineDiff int) {
	logger.Info("[MONITORING] [%s] Request completed: %s %s [%d] in %v",
		ctx.RequestID, ctx.Method, ctx.Path, statusCode, ctx.ResponseTime)

	logger.Debug("[MONITORING] [%s] Memory after: %d bytes (diff: %+d), Goroutines: %d (diff: %+d)",
		ctx.RequestID, ctx.MemoryAfter, memoryDiff, ctx.GoroutinesAfter, goroutineDiff)
}

func (ctx *RequestContext) logComponentBreakdown() {
	tsTime := ctx.calculateDuration(ctx.TSStartTime, ctx.TSEndTime)
	dbTime := ctx.calculateDuration(ctx.DBStartTime, ctx.DBEndTime)
	parsingTime := ctx.calculateDuration(ctx.ParsingStartTime, ctx.ParsingEndTime)
	responseTime := ctx.calculateDuration(ctx.ResponseStartTime, ctx.ResponseEndTime)
	otherTime := ctx.ResponseTime - tsTime - dbTime - parsingTime - responseTime

	logger.Info("[MONITORING] [%s] Breakdown - TS: %v, DB: %v, Parsing: %v, Response: %v, Other: %v",
		ctx.RequestID, tsTime, dbTime, parsingTime, responseTime, otherTime)
}

func (ctx *RequestContext) calculateDuration(start, end time.Time) time.Duration {
	if start.IsZero() || end.IsZero() {
		return 0
	}
	return end.Sub(start)
}

func (ctx *RequestContext) logPerformanceWarnings(memoryDiff int64, goroutineDiff int) {
	ctx.warnForSlowRequests()
	ctx.warnForMemoryLeaks(memoryDiff)
	ctx.warnForGoroutineLeaks(goroutineDiff)
}

func (ctx *RequestContext) warnForSlowRequests() {
	if ctx.ResponseTime > 500*time.Millisecond {
		logger.Warn("[MONITORING] [%s] SLOW REQUEST: %v (>500ms) - Consider optimization",
			ctx.RequestID, ctx.ResponseTime)
	}
}

func (ctx *RequestContext) warnForMemoryLeaks(memoryDiff int64) {
	if memoryDiff > 1024*1024 { // More than 1MB increase
		logger.Warn("[MONITORING] [%s] HIGH MEMORY USAGE: %+d bytes allocated",
			ctx.RequestID, memoryDiff)
	}
}

func (ctx *RequestContext) warnForGoroutineLeaks(goroutineDiff int) {
	if goroutineDiff <= 0 {
		return
	}

	// Use the goroutine debugger to identify the source and filter background operations
	hasActualLeaks := false
	if ctx.goroutineDebugger != nil {
		hasActualLeaks = ctx.goroutineDebugger.CheckForLeaks(ctx.RequestID)
	}

	// Only warn if there are actual leaks (not just background goroutines)
	if hasActualLeaks {
		logger.Warn("[MONITORING] [%s] GOROUTINE LEAK: %+d goroutines created (excluding background operations)",
			ctx.RequestID, goroutineDiff)
	} else {
		logger.Debug("[MONITORING] [%s] Goroutine increase: %+d (all expected background operations)",
			ctx.RequestID, goroutineDiff)
	}
}

// calculateMemoryDiff safely calculates the difference between memory values
// without risk of integer overflow during conversion.
func calculateMemoryDiff(before, after uint64) int64 {
	const maxSafeUint64 = uint64(1<<63 - 1) // Maximum value that fits in int64

	if after >= before {
		diff := after - before
		if diff > maxSafeUint64 {
			return 1<<63 - 1 // Return max int64 to prevent overflow
		}
		return int64(diff)
	}

	diff := before - after
	if diff > maxSafeUint64 {
		return -(1<<63 - 1) // Return min int64 to prevent overflow
	}
	return -int64(diff)
}

// GetMetrics returns a snapshot of current performance metrics.
//
// This function provides thread-safe access to the current performance metrics
// by creating a copy of the global metrics with atomic operations. The returned
// metrics represent the state at the time of the call.
//
// Returns a Metrics struct containing current performance statistics.
func GetMetrics() *Metrics {
	return &Metrics{
		RequestCount:        atomic.LoadInt64(&globalMetrics.RequestCount),
		TotalResponseTime:   atomic.LoadInt64(&globalMetrics.TotalResponseTime),
		AverageResponseTime: globalMetrics.AverageResponseTime,
		MaxResponseTime:     atomic.LoadInt64(&globalMetrics.MaxResponseTime),
		MinResponseTime:     atomic.LoadInt64(&globalMetrics.MinResponseTime),
		TSExecutionTime:     atomic.LoadInt64(&globalMetrics.TSExecutionTime),
		DBExecutionTime:     atomic.LoadInt64(&globalMetrics.DBExecutionTime),
		ActiveRequests:      atomic.LoadInt64(&globalMetrics.ActiveRequests),
	}
}

// LogMetrics logs comprehensive performance metrics to the console.
//
// This function outputs detailed performance information including request counts,
// response times, component-specific timing, memory usage, and goroutine counts.
// It's useful for periodic monitoring and debugging performance issues.
func LogMetrics() {
	metrics := GetMetrics()

	logger.Info("[MONITORING] Total Requests: %d", metrics.RequestCount)
	logger.Info("[MONITORING] Average Response Time: %.2fms", metrics.AverageResponseTime)
	logger.Info("[MONITORING] Min/Max Response Time: %dms/%dms", metrics.MinResponseTime, metrics.MaxResponseTime)
	logger.Info("[MONITORING] Total TS Execution Time: %dms", metrics.TSExecutionTime)
	logger.Info("[MONITORING] Total DB Execution Time: %dms", metrics.DBExecutionTime)
	logger.Info("[MONITORING] Active Requests: %d", metrics.ActiveRequests)

	// Memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	logger.Info("[MONITORING] Memory - Alloc: %d KB, Sys: %d KB, NumGC: %d",
		memStats.Alloc/1024, memStats.Sys/1024, memStats.NumGC)
	logger.Info("[MONITORING] Goroutines: %d", runtime.NumGoroutine())
}

// StartPeriodicMetrics starts periodic logging of performance metrics.
//
// This function initializes a background goroutine that logs performance metrics
// at regular intervals. It uses sync.Once to ensure metrics logging is started
// only once, even if called multiple times.
//
// Parameters:
//   - interval: Duration between metric logging events
//
// The periodic logging continues until the application terminates.
func StartPeriodicMetrics(interval time.Duration) {
	metricsStarted.Do(func() {
		logger.Info("[MONITORING] Starting periodic metrics logging every %v", interval)
		go func() {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for range ticker.C {
				LogMetrics()
			}
		}()
	})
}
