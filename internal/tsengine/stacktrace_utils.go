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

// Package tsengine provides comprehensive stacktrace utilities for error debugging.
package tsengine

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
)

// StackTraceFrame represents a single frame in a stacktrace.
type StackTraceFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Package  string `json:"package"`
}

// DetailedError represents an enhanced error with full debugging information.
type DetailedError struct {
	OriginalError    error             `json:"original_error"`
	Message          string            `json:"message"`
	Timestamp        time.Time         `json:"timestamp"`
	GoStackTrace     []StackTraceFrame `json:"go_stacktrace"`
	JSStackTrace     string            `json:"js_stacktrace,omitempty"`
	Context          map[string]any    `json:"context,omitempty"`
	ErrorType        string            `json:"error_type"`
	SourceLocation   string            `json:"source_location,omitempty"`
	RuntimeDebugInfo string            `json:"runtime_debug_info,omitempty"`
}

// Error implements the error interface.
func (de *DetailedError) Error() string {
	return de.Message
}

// String provides a comprehensive string representation of the error.
func (de *DetailedError) String() string {
	var sb strings.Builder

	sb.WriteString("=== TurboScript Detailed Error Report ===\n")
	sb.WriteString(fmt.Sprintf("Timestamp: %s\n", de.Timestamp.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Error Type: %s\n", de.ErrorType))
	sb.WriteString(fmt.Sprintf("Message: %s\n", de.Message))

	if de.SourceLocation != "" {
		sb.WriteString(fmt.Sprintf("Source Location: %s\n", de.SourceLocation))
	}

	if de.OriginalError != nil {
		sb.WriteString(fmt.Sprintf("Original Error: %v\n", de.OriginalError))
	}

	// Add Go stacktrace
	sb.WriteString("\n--- Go Stacktrace ---\n")
	for i, frame := range de.GoStackTrace {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, frame.Function))
		sb.WriteString(fmt.Sprintf("   %s:%d\n", frame.File, frame.Line))
		if frame.Package != "" {
			sb.WriteString(fmt.Sprintf("   Package: %s\n", frame.Package))
		}
		sb.WriteString("\n")
	}

	// Add JavaScript stacktrace if available
	if de.JSStackTrace != "" {
		sb.WriteString("--- JavaScript Stacktrace ---\n")
		sb.WriteString(de.JSStackTrace)
		sb.WriteString("\n")
	}

	// Add context information
	if len(de.Context) > 0 {
		sb.WriteString("--- Context Information ---\n")
		for key, value := range de.Context {
			sb.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
		sb.WriteString("\n")
	}

	// Add runtime debug info if available
	if de.RuntimeDebugInfo != "" {
		sb.WriteString("--- Runtime Debug Information ---\n")
		sb.WriteString(de.RuntimeDebugInfo)
		sb.WriteString("\n")
	}

	sb.WriteString("=== End Error Report ===")

	return sb.String()
}

// StackTraceUtils provides utilities for creating detailed error reports with stacktraces.
type StackTraceUtils struct{}

// NewStackTraceUtils creates a new stacktrace utilities instance.
func NewStackTraceUtils() *StackTraceUtils {
	return &StackTraceUtils{}
}

// WrapError wraps an error with comprehensive debugging information.
func (stu *StackTraceUtils) WrapError(err error, errorType, message string, context map[string]any) *DetailedError {
	return stu.WrapErrorWithSource(err, errorType, message, "", context)
}

// WrapErrorWithSource wraps an error with source location and comprehensive debugging information.
func (stu *StackTraceUtils) WrapErrorWithSource(err error, errorType, message, sourceLocation string, context map[string]any) *DetailedError {
	stackTrace := stu.captureGoStackTrace(3) // Skip this function and the wrapper functions

	detailedErr := &DetailedError{
		OriginalError:    err,
		Message:          message,
		Timestamp:        time.Now(),
		GoStackTrace:     stackTrace,
		Context:          context,
		ErrorType:        errorType,
		SourceLocation:   sourceLocation,
		RuntimeDebugInfo: stu.captureRuntimeDebugInfo(),
	}

	// Try to extract JavaScript stack if the error contains it
	if err != nil {
		errStr := err.Error()
		if jsStack := stu.extractJavaScriptStack(errStr); jsStack != "" {
			detailedErr.JSStackTrace = jsStack
		}
	}

	return detailedErr
}

// WrapErrorf wraps an error with formatted message and debugging information.
func (stu *StackTraceUtils) WrapErrorf(err error, errorType string, format string, args ...any) *DetailedError {
	message := fmt.Sprintf(format, args...)
	return stu.WrapError(err, errorType, message, nil)
}

// NewError creates a new detailed error without wrapping an existing error.
func (stu *StackTraceUtils) NewError(errorType, message string, context map[string]any) *DetailedError {
	return stu.WrapError(nil, errorType, message, context)
}

// NewErrorf creates a new detailed error with formatted message.
func (stu *StackTraceUtils) NewErrorf(errorType, format string, args ...any) *DetailedError {
	message := fmt.Sprintf(format, args...)
	return stu.NewError(errorType, message, nil)
}

// LogAndWrapError logs the error with full details and returns a wrapped error.
func (stu *StackTraceUtils) LogAndWrapError(err error, errorType, message string, context map[string]any) *DetailedError {
	detailedErr := stu.WrapError(err, errorType, message, context)
	stu.logDetailedError(detailedErr)
	return detailedErr
}

// LogAndWrapErrorWithSource logs the error with source location and returns a wrapped error.
func (stu *StackTraceUtils) LogAndWrapErrorWithSource(err error, errorType, message, sourceLocation string, context map[string]any) *DetailedError {
	detailedErr := stu.WrapErrorWithSource(err, errorType, message, sourceLocation, context)
	stu.logDetailedError(detailedErr)
	return detailedErr
}

// captureGoStackTrace captures the current Go stacktrace.
func (stu *StackTraceUtils) captureGoStackTrace(skip int) []StackTraceFrame {
	var frames []StackTraceFrame

	// Get program counters for stack frames
	pc := make([]uintptr, 50) // Capture up to 50 frames
	n := runtime.Callers(skip, pc)
	pc = pc[:n]

	frameIterator := runtime.CallersFrames(pc)
	for {
		frame, more := frameIterator.Next()

		// Extract package name from function
		packageName := ""
		if idx := strings.LastIndex(frame.Function, "/"); idx != -1 {
			if dotIdx := strings.Index(frame.Function[idx+1:], "."); dotIdx != -1 {
				packageName = frame.Function[idx+1 : idx+1+dotIdx]
			}
		}

		frames = append(frames, StackTraceFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
			Package:  packageName,
		})

		if !more {
			break
		}
	}

	return frames
}

// captureRuntimeDebugInfo captures additional runtime debugging information.
func (stu *StackTraceUtils) captureRuntimeDebugInfo() string {
	var sb strings.Builder

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	sb.WriteString(fmt.Sprintf("Goroutines: %d\n", runtime.NumGoroutine()))
	sb.WriteString(fmt.Sprintf("Memory Allocated: %d KB\n", memStats.Alloc/1024))
	sb.WriteString(fmt.Sprintf("Total Allocations: %d\n", memStats.TotalAlloc))
	sb.WriteString(fmt.Sprintf("System Memory: %d KB\n", memStats.Sys/1024))
	sb.WriteString(fmt.Sprintf("GC Cycles: %d\n", memStats.NumGC))

	// Get build info if available
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		sb.WriteString(fmt.Sprintf("Go Version: %s\n", buildInfo.GoVersion))
		sb.WriteString(fmt.Sprintf("Main Module: %s\n", buildInfo.Main.Path))
		if buildInfo.Main.Version != "" {
			sb.WriteString(fmt.Sprintf("Module Version: %s\n", buildInfo.Main.Version))
		}
	}

	return sb.String()
}

// extractJavaScriptStack attempts to extract JavaScript stack trace from error message.
func (stu *StackTraceUtils) extractJavaScriptStack(errorMessage string) string {
	// Look for common JavaScript stack trace patterns
	patterns := []string{
		"at ",        // V8 style stack traces
		"Error:",     // Standard error format
		"TypeError:", // Common JS error
		"ReferenceError:",
		"SyntaxError:",
	}

	for _, pattern := range patterns {
		if strings.Contains(errorMessage, pattern) {
			// Try to extract the stack portion
			lines := strings.Split(errorMessage, "\n")
			var stackLines []string
			capturing := false

			for _, line := range lines {
				if strings.Contains(line, pattern) {
					capturing = true
				}
				if capturing {
					stackLines = append(stackLines, line)
				}
			}

			if len(stackLines) > 0 {
				return strings.Join(stackLines, "\n")
			}
		}
	}

	return ""
}

// logDetailedError logs the detailed error with appropriate level.
func (stu *StackTraceUtils) logDetailedError(detailedErr *DetailedError) {
	// Log a concise version first
	logger.Error("[%s] %s", detailedErr.ErrorType, detailedErr.Message)

	if detailedErr.OriginalError != nil {
		logger.Error("Original error: %v", detailedErr.OriginalError)
	}

	if detailedErr.SourceLocation != "" {
		logger.Error("Source: %s", detailedErr.SourceLocation)
	}

	// Log context if available
	if len(detailedErr.Context) > 0 {
		for key, value := range detailedErr.Context {
			logger.Debug("Context %s: %v", key, value)
		}
	}

	// Log a subset of the Go stacktrace (first few frames)
	if len(detailedErr.GoStackTrace) > 0 {
		logger.Debug("Go stacktrace (top 5 frames):")
		maxFrames := len(detailedErr.GoStackTrace)
		if maxFrames > 5 {
			maxFrames = 5
		}
		for i := 0; i < maxFrames; i++ {
			frame := detailedErr.GoStackTrace[i]
			logger.Debug("  %d. %s (%s:%d)", i+1, frame.Function, frame.File, frame.Line)
		}
	}

	// Log JavaScript stack if available
	if detailedErr.JSStackTrace != "" {
		logger.Debug("JavaScript stacktrace:")
		jsLines := strings.Split(detailedErr.JSStackTrace, "\n")
		for i, line := range jsLines {
			if i >= 10 { // Limit JS stack output
				break
			}
			logger.Debug("  %s", line)
		}
	}

	// For debug mode, log the full detailed error
	logger.Debug("Full error report available in debug output")
}

// Common error types for TurboScript.
const (
	ErrorTypeAsyncExecution     = "AsyncExecution"
	ErrorTypeJSCompilation      = "JSCompilation"
	ErrorTypeHandleFunction     = "HandleFunction"
	ErrorTypeDatabaseQuery      = "DatabaseQuery"
	ErrorTypeJobDispatch        = "JobDispatch"
	ErrorTypeBroadcast          = "Broadcast"
	ErrorTypeEmailService       = "EmailService"
	ErrorTypeCacheOperation     = "CacheOperation"
	ErrorTypeMarkdownProcessing = "MarkdownProcessing"
	ErrorTypeAuthVerification   = "AuthVerification"
	ErrorTypeConfigurationLoad  = "ConfigurationLoad"
	ErrorTypeFileSystem         = "FileSystem"
	ErrorTypeNetworkOperation   = "NetworkOperation"
	ErrorTypeValidation         = "Validation"
	ErrorTypeTimeout            = "Timeout"
	ErrorTypeResourceExhaustion = "ResourceExhaustion"
)
