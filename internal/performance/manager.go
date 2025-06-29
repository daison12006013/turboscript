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

// Package performance provides performance monitoring and profiling capabilities for TurboScript.
//
// This package implements comprehensive performance monitoring including CPU profiling,
// memory profiling, goroutine analysis, and runtime metrics collection. It integrates
// with Go's built-in pprof package to provide detailed performance insights.
//
// Key Features:
//   - CPU profiling with configurable duration
//   - Memory heap and allocation profiling
//   - Goroutine monitoring and analysis
//   - Runtime statistics collection
//   - HTTP endpoint for profile data retrieval
//   - Integration with performance monitoring tools
//
// The performance manager can be enabled through configuration and provides
// real-time insights into application performance characteristics.
package performance

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
)

// ProfileManager handles profiling operations and performance data collection.
//
// The ProfileManager provides a centralized interface for collecting various types
// of performance profiles including CPU usage, memory allocation, and goroutine
// analysis. It communicates with Go's built-in pprof server to gather profile data.
type ProfileManager struct {
	ProfileServerURL string       // URL of the pprof server endpoint
	httpClient       *http.Client // Optimized HTTP client for profile requests
}

// createOptimizedHTTPClient creates an HTTP client optimized for profile requests.
func createOptimizedHTTPClient(timeoutSeconds int) *http.Client {
	return &http.Client{
		Timeout: time.Duration(timeoutSeconds+10) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    time.Duration(timeoutSeconds) * time.Second,
			DisableCompression: false,
		},
	}
}

// NewProfileManager creates a new ProfileManager instance.
//
// If profileServerURL is empty, it defaults to "http://localhost:6060".
// The ProfileManager assumes that a pprof server is running at the specified URL.
func NewProfileManager(profileServerURL string) *ProfileManager {
	if profileServerURL == "" {
		profileServerURL = "http://localhost:6060"
	}
	return &ProfileManager{
		ProfileServerURL: profileServerURL,
		httpClient:       createOptimizedHTTPClient(30),
	}
}

// getTimeoutSeconds returns a reasonable timeout for HTTP requests.
func (pm *ProfileManager) getTimeoutSeconds() int {
	return 30 // Default timeout for profiling operations
}

// GetCPUProfile fetches a CPU profile for the specified duration.
//
// This method collects CPU profiling data by making an HTTP request to the
// pprof endpoint. The profile data shows where CPU time is being spent,
// helping identify performance bottlenecks.
//
// Parameters:
//   - durationSeconds: Duration to collect CPU profile data (defaults to 30 if <= 0)
//
// Returns an error if the profile collection fails or the server is unreachable.
func (pm *ProfileManager) GetCPUProfile(durationSeconds int) error {
	if durationSeconds <= 0 {
		durationSeconds = 30
	}

	url := fmt.Sprintf("%s/debug/pprof/profile?seconds=%d", pm.ProfileServerURL, durationSeconds)
	logger.Info("[PROFILE] Collecting CPU profile for %d seconds from %s", durationSeconds, url)

	// Update client timeout for this specific request if needed
	if durationSeconds > 30 {
		pm.httpClient = createOptimizedHTTPClient(durationSeconds)
	}

	resp, err := pm.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch CPU profile: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Error("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CPU profile request failed with status: %d", resp.StatusCode)
	}

	logger.Info("[PROFILE] CPU profile collection completed successfully")
	logger.Info("[PROFILE] To analyze: curl %s > cpu.prof && go tool pprof cpu.prof", url)
	return nil
}

// GetMemoryProfile fetches a memory heap profile.
//
// This method collects memory profiling data showing heap allocations,
// helping identify memory leaks and excessive memory usage patterns.
// The profile includes information about allocated objects and their sizes.
//
// Returns an error if the profile collection fails or the server is unreachable.
func (pm *ProfileManager) GetMemoryProfile() error {
	url := fmt.Sprintf("%s/debug/pprof/heap", pm.ProfileServerURL)
	logger.Info("[PROFILE] Collecting memory profile from %s", url)

	client := &http.Client{Timeout: time.Duration(pm.getTimeoutSeconds()) * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch memory profile: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Error("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("memory profile request failed with status: %d", resp.StatusCode)
	}

	logger.Info("[PROFILE] Memory profile collection completed successfully")
	logger.Info("[PROFILE] To analyze: curl %s > mem.prof && go tool pprof mem.prof", url)
	return nil
}

// GetGoroutineProfile fetches a goroutine profile.
//
// This method collects information about all active goroutines in the application,
// including their stack traces and current state. This is useful for debugging
// goroutine leaks and understanding concurrency patterns.
//
// Returns an error if the profile collection fails or the server is unreachable.
func (pm *ProfileManager) GetGoroutineProfile() error {
	url := fmt.Sprintf("%s/debug/pprof/goroutine", pm.ProfileServerURL)
	logger.Info("[PROFILE] Collecting goroutine profile from %s", url)

	client := &http.Client{Timeout: time.Duration(pm.getTimeoutSeconds()) * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch goroutine profile: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Error("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("goroutine profile request failed with status: %d", resp.StatusCode)
	}

	logger.Info("[PROFILE] Goroutine profile collection completed successfully")
	logger.Info("[PROFILE] To analyze: curl %s > goroutine.prof && go tool pprof goroutine.prof", url)
	return nil
}

// GetTrace fetches an execution trace for the specified duration.
//
// This method collects detailed execution traces that show goroutine scheduling,
// garbage collection events, and other runtime activities. Traces are essential
// for understanding concurrency bottlenecks and runtime behavior.
//
// Parameters:
//   - durationSeconds: Duration to collect trace data (defaults to 5 if <= 0)
//
// Returns an error if the trace collection fails or the server is unreachable.
func (pm *ProfileManager) GetTrace(durationSeconds int) error {
	if durationSeconds <= 0 {
		durationSeconds = 5
	}

	url := fmt.Sprintf("%s/debug/pprof/trace?seconds=%d", pm.ProfileServerURL, durationSeconds)
	logger.Info("[PROFILE] Collecting execution trace for %d seconds from %s", durationSeconds, url)

	client := &http.Client{Timeout: time.Duration(durationSeconds+10) * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch execution trace: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Error("Failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("execution trace request failed with status: %d", resp.StatusCode)
	}

	logger.Info("[PROFILE] Execution trace collection completed successfully")
	logger.Info("[PROFILE] To analyze: curl %s > trace.out && go tool trace trace.out", url)
	return nil
}

// PrintMetricsJSON prints current performance metrics in JSON format.
//
// This function outputs a comprehensive set of performance metrics as formatted JSON,
// making it easy to integrate with monitoring systems or parse programmatically.
// The output includes request counts, response times, and component-specific metrics.
func PrintMetricsJSON() {
	metrics := GetMetrics()
	jsonData, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal metrics to JSON: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}

// PrintDetailedReport prints a comprehensive performance report with analysis.
//
// This function outputs a detailed human-readable performance report that includes:
//   - Request statistics (count, active requests)
//   - Response time analysis (average, min, max)
//   - Component-specific timing (TypeScript, Database)
//   - Performance recommendations based on current metrics
//   - Profiling command suggestions for further analysis
//
// The report is designed to help developers identify performance bottlenecks
// and optimization opportunities in their TurboScript applications.
func PrintDetailedReport() {
	metrics := GetMetrics()

	printReportHeader(metrics)
	printResponseTimes(metrics)
	printComponentTimes(metrics)
	printRecommendations(metrics)
	printProfilingCommands()
}

func printReportHeader(metrics *Metrics) {
	fmt.Println("=== TurboScript Performance Report ===")
	fmt.Printf("Total Requests: %d\n", metrics.RequestCount)
	fmt.Printf("Active Requests: %d\n", metrics.ActiveRequests)
	fmt.Println()
}

func printResponseTimes(metrics *Metrics) {
	fmt.Println("Response Times:")
	fmt.Printf("  Average: %.2f ms\n", metrics.AverageResponseTime)
	fmt.Printf("  Minimum: %d ms\n", metrics.MinResponseTime)
	fmt.Printf("  Maximum: %d ms\n", metrics.MaxResponseTime)
	fmt.Println()
}

func printComponentTimes(metrics *Metrics) {
	fmt.Println("Component Times:")
	fmt.Printf("  Total TypeScript Execution: %d ms\n", metrics.TSExecutionTime)
	fmt.Printf("  Total Database Execution: %d ms\n", metrics.DBExecutionTime)

	if metrics.RequestCount > 0 {
		avgTSTime := float64(metrics.TSExecutionTime) / float64(metrics.RequestCount)
		avgDBTime := float64(metrics.DBExecutionTime) / float64(metrics.RequestCount)
		fmt.Printf("  Average TypeScript per request: %.2f ms\n", avgTSTime)
		fmt.Printf("  Average Database per request: %.2f ms\n", avgDBTime)
	}
	fmt.Println()
}

func printRecommendations(metrics *Metrics) {
	fmt.Println("Performance Recommendations:")

	if metrics.AverageResponseTime > 500 {
		fmt.Println("  ⚠️  High average response time (>500ms) - consider optimization")
	}

	if metrics.RequestCount > 0 {
		avgTSTime := float64(metrics.TSExecutionTime) / float64(metrics.RequestCount)
		avgDBTime := float64(metrics.DBExecutionTime) / float64(metrics.RequestCount)

		if avgTSTime > 100 {
			fmt.Println("  ⚠️  High TypeScript execution time - consider code optimization")
		}

		if avgDBTime > 200 {
			fmt.Println("  ⚠️  High database execution time - consider query optimization or indexing")
		}

		printBottleneckRecommendations(avgTSTime, avgDBTime)
	}

	if metrics.ActiveRequests > 10 {
		fmt.Printf("  ⚠️  High number of active requests (%d) - possible concurrency issues\n", metrics.ActiveRequests)
	}
	fmt.Println()
}

func printBottleneckRecommendations(avgTSTime, avgDBTime float64) {
	if avgTSTime > avgDBTime*2 {
		fmt.Println("  💡 TypeScript execution is bottleneck - focus on TS code optimization")
	} else if avgDBTime > avgTSTime*2 {
		fmt.Println("  💡 Database execution is bottleneck - focus on query optimization")
	}
}

func printProfilingCommands() {
	printProfilingCommandsWithPort("6060")
}

func printProfilingCommandsWithPort(port string) {
	fmt.Println("Profiling Commands:")
	fmt.Printf("  CPU Profile: curl http://localhost:%s/debug/pprof/profile > cpu.prof\n", port)
	fmt.Printf("  Memory Profile: curl http://localhost:%s/debug/pprof/heap > mem.prof\n", port)
	fmt.Printf("  Goroutine Profile: curl http://localhost:%s/debug/pprof/goroutine > goroutine.prof\n", port)
	fmt.Println("  Analyze: go tool pprof [profile-file]")
	fmt.Println("================================")
}
