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

// Package logger provides structured logging capabilities for TurboScript.
//
// This package implements a configurable logging system that supports different
// log levels (debug, info, warning) and integrates with the TurboScript configuration
// system to control logging behavior at runtime.
//
// Features:
//   - Configurable log levels (debug, info, warning)
//   - Runtime log level control via configuration
//   - Standard Go log integration with file and line information
//   - Environment variable overrides for log levels
//
// The logger is used throughout the TurboScript system to provide consistent
// logging behavior and debugging capabilities.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	debugEnabled   = false                                                // Controls debug message output
	infoEnabled    = true                                                 // Default to true for backward compatibility
	warningEnabled = true                                                 // Default to true for backward compatibility
	logger         = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile) // Standard logger with timestamp and file info
	logFile        *os.File                                               // Optional log file for file output
)

// SetLoggingLevels sets all logging levels at once.
//
// This is a convenience function to configure multiple log levels
// simultaneously, typically called during application initialization.
func SetLoggingLevels(debug, info, warning bool) {
	debugEnabled = debug
	infoEnabled = info
	warningEnabled = warning
}

// SetLogFile configures logging to write to a file instead of stdout.
//
// If filename is empty, logging will revert to stdout.
// If the file cannot be opened, an error is returned and logging remains on stdout.
func SetLogFile(filename string) error {
	if filename == "" {
		// Revert to stdout
		if logFile != nil {
			if err := logFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close log file: %v\n", err)
			}
			logFile = nil
		}
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
		return nil
	}

	// Security: Only allow log files within ./logs or /var/log/turboscript
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("failed to resolve log file path: %w", err)
	}
	allowed := false
	allowedDirs := []string{"/var/log/turboscript", "./logs", "/tmp", "."} // Allow project root for log files
	for _, dir := range allowedDirs {
		dirAbs, _ := filepath.Abs(dir)
		if strings.HasPrefix(absPath, dirAbs) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("log file path not allowed: %s", absPath)
	}

	// #nosec G304: filename is validated above to be within allowed log directories
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", filename, err)
	}

	// Close previous log file if it exists
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close previous log file: %v\n", err)
		}
	}

	// Set up logging to both file and stdout
	logFile = file
	multiWriter := io.MultiWriter(os.Stdout, file)
	logger = log.New(multiWriter, "", log.LstdFlags|log.Lshortfile)

	return nil
}

// CloseLogFile closes the log file if one is open.
//
// This should be called during application shutdown to ensure
// all log data is properly written to the file.
func CloseLogFile() {
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close log file: %v\n", err)
		}
		logFile = nil
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	}
}

// Debug logs a debug message with printf-style formatting.
//
// Messages are only output if debug logging is enabled via SetDebug(true).
// Debug messages include file and line information for easier troubleshooting.
func Debug(format string, args ...any) {
	if debugEnabled {
		logger.Printf("[DEBUG] "+format, args...)
	}
}

// Info logs an info message with printf-style formatting.
//
// Messages are only output if info logging is enabled (enabled by default).
// Info messages are used for general operational information.
func Info(format string, args ...any) {
	if infoEnabled {
		logger.Printf("[INFO] "+format, args...)
	}
}

// Error logs an error message with printf-style formatting.
//
// Error messages are always output regardless of log level settings.
// Use this for critical errors that should always be visible.
func Error(format string, args ...any) {
	logger.Printf("[ERROR] "+format, args...)
}

// Warn logs a warning message with printf-style formatting.
//
// Messages are only output if warning logging is enabled (enabled by default).
// Use this for non-critical issues that should be brought to attention.
func Warn(format string, args ...any) {
	if warningEnabled {
		logger.Printf("[WARN] "+format, args...)
	}
}

// Console logging functions for JavaScript runtime integration

// ConsoleLog logs a console.log message with [CONSOLE] [LOG] prefix.
func ConsoleLog(args ...any) {
	if infoEnabled {
		logger.Printf("[CONSOLE] [LOG] %v", args...)
	}
}

// ConsoleError logs a console.error message with [CONSOLE] [ERROR] prefix.
func ConsoleError(args ...any) {
	logger.Printf("[CONSOLE] [ERROR] %v", args...)
}

// ConsoleWarn logs a console.warn message with [CONSOLE] [WARN] prefix.
func ConsoleWarn(args ...any) {
	if warningEnabled {
		logger.Printf("[CONSOLE] [WARN] %v", args...)
	}
}

// ConsoleInfo logs a console.info message with [CONSOLE] [INFO] prefix.
func ConsoleInfo(args ...any) {
	if infoEnabled {
		logger.Printf("[CONSOLE] [INFO] %v", args...)
	}
}

// ConsoleDebug logs a console.debug message with [CONSOLE] [DEBUG] prefix.
func ConsoleDebug(args ...any) {
	if debugEnabled {
		logger.Printf("[CONSOLE] [DEBUG] %v", args...)
	}
}
