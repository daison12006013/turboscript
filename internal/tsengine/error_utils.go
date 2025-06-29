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

// Package tsengine provides error handling utilities for TypeScript execution.
package tsengine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daison12006013/turboscript/internal/logger"
)

// Error type constants for TypeScript execution.
const (
	ErrorTypeUnknown    = "Unknown TypeScript error"
	ErrorTypeExecution  = "TypeScript execution error"
	ErrorTypeResponse   = "TypeScript response handler error"
	ErrorTypeDataSource = "TypeScript data source error"
)

// ErrorUtils handles various types of TypeScript execution errors.
type ErrorUtils struct{}

// NewErrorUtils creates a new error utilities instance.
func NewErrorUtils() *ErrorUtils {
	return &ErrorUtils{}
}

// CheckForTSExecutionError checks if the result string contains a TypeScript execution error.
func (eu *ErrorUtils) CheckForTSExecutionError(resultStr, tsPath string) error {
	var errorCheck map[string]any
	if err := json.Unmarshal([]byte(resultStr), &errorCheck); err != nil {
		return fmt.Errorf("failed to parse TypeScript result as JSON: %w", err)
	}

	if tsError, hasError := errorCheck["__ts_error"]; hasError {
		return eu.processTSError(tsError, tsPath)
	}

	return nil
}

// CheckForExecutionError checks if the result string contains an execution error.
func (eu *ErrorUtils) CheckForExecutionError(resultStr, tsPath string) error {
	var errorCheck map[string]any
	if err := json.Unmarshal([]byte(resultStr), &errorCheck); err != nil {
		return fmt.Errorf("failed to parse response handlers result as JSON: %w", err)
	}

	if errorData, hasError := errorCheck["error"]; hasError {
		return eu.ProcessResponseHandlerError(errorData, tsPath)
	}

	return nil
}

// CheckForDataSourcesError checks if the result string contains a data sources error.
func (eu *ErrorUtils) CheckForDataSourcesError(resultStr, tsPath string) error {
	if !strings.Contains(resultStr, "__ts_error") {
		return nil
	}

	var errorCheck map[string]any
	if err := json.Unmarshal([]byte(resultStr), &errorCheck); err != nil {
		return nil
	}

	tsError, hasError := errorCheck["__ts_error"]
	if !hasError {
		return nil
	}

	return eu.processDataSourceError(tsError, tsPath)
}

// ProcessHandleError processes and formats errors from the handle function.
func (eu *ErrorUtils) ProcessHandleError(errorData any, tsPath string) error {
	if errorMap, ok := errorData.(map[string]any); ok {
		message := ErrorTypeUnknown
		if msg, exists := errorMap["message"]; exists {
			if msgStr, ok := msg.(string); ok {
				message = msgStr
			}
		}
		return fmt.Errorf("TypeScript handle function failed in '%s': %s", tsPath, message)
	}
	return fmt.Errorf("TypeScript handle function failed in '%s' with unknown error format", tsPath)
}

// ProcessResponseHandlerError processes and formats errors from the response handler.
func (eu *ErrorUtils) ProcessResponseHandlerError(errorData any, tsPath string) error {
	if errorMap, ok := errorData.(map[string]any); ok {
		message := ErrorTypeUnknown
		if msg, exists := errorMap["message"]; exists {
			if msgStr, ok := msg.(string); ok {
				message = msgStr
			}
		}
		return fmt.Errorf("TypeScript response handlers failed in '%s': %s", tsPath, message)
	}
	return fmt.Errorf("TypeScript response handlers failed in '%s' with unknown error format", tsPath)
}

func (eu *ErrorUtils) processTSError(tsError any, tsPath string) error {
	errorMap, ok := tsError.(map[string]any)
	if !ok {
		return fmt.Errorf("TypeScript execution failed in '%s' with unknown error format", tsPath)
	}

	message := eu.extractErrorMessage(errorMap)
	stack := eu.extractErrorStack(errorMap)

	logger.Error("TypeScript execution failed in '%s': %s", tsPath, message)
	if stack != "" {
		logger.Error("Stack trace: %s", stack)
	}

	return fmt.Errorf("TypeScript execution failed in '%s': %s", tsPath, message)
}

func (eu *ErrorUtils) processDataSourceError(tsError any, tsPath string) error {
	errorMap, ok := tsError.(map[string]any)
	if !ok {
		return fmt.Errorf("TypeScript execution failed in '%s' with unknown error format", tsPath)
	}

	message := eu.extractErrorMessage(errorMap)
	return fmt.Errorf("TypeScript execution failed in '%s': %s", tsPath, message)
}

func (eu *ErrorUtils) extractErrorMessage(errorMap map[string]any) string {
	msg, exists := errorMap["message"]
	if !exists {
		return ErrorTypeUnknown
	}

	msgStr, ok := msg.(string)
	if !ok {
		return ErrorTypeUnknown
	}

	return msgStr
}

func (eu *ErrorUtils) extractErrorStack(errorMap map[string]any) string {
	st, exists := errorMap["stack"]
	if !exists {
		return ""
	}

	stStr, ok := st.(string)
	if !ok {
		return ""
	}

	return stStr
}
