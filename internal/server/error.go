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

package server

import (
	"encoding/json"
	"fmt"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/tsengine"
	"github.com/valyala/fasthttp"
)

// sendPanicErrorResponse sends a JSON error response when a panic occurs.
func (s *Server) sendPanicErrorResponse(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusInternalServerError)
	ctx.SetContentType("application/json")

	errorResponse := map[string]any{
		"status": "error",
		"error": map[string]any{
			"type":    "internal_panic",
			"message": "An internal error occurred. The request has been logged.",
		},
	}

	jsonBytes, err := json.Marshal(errorResponse)
	if err != nil {
		s.writeErrorString(ctx, `{"status":"error","error":{"type":"internal_panic","message":"Critical error occurred"}}`)
		return
	}

	s.writeErrorBytes(ctx, jsonBytes)
}

// writeErrorBytes writes JSON bytes to the response context with compression support.
func (s *Server) writeErrorBytes(ctx *fasthttp.RequestCtx, jsonBytes []byte) {
	if writeErr := s.writeCompressedResponse(ctx, jsonBytes); writeErr != nil {
		logger.Error("Failed to write panic error response: %v", writeErr)
	}
}

// writeErrorString writes an error string to the response context with compression support.
func (s *Server) writeErrorString(ctx *fasthttp.RequestCtx, errorStr string) {
	if writeErr := s.writeCompressedResponse(ctx, []byte(errorStr)); writeErr != nil {
		logger.Error("Failed to write critical error response: %v", writeErr)
	}
}

// sendError sends a detailed error response with enhanced debugging information.
func (s *Server) sendError(ctx *fasthttp.RequestCtx, errorType, message string) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusInternalServerError)

	errorResponse := map[string]any{
		"status": "error",
		"error": map[string]any{
			"type":    errorType,
			"message": message,
		},
	}

	responseData, err := json.Marshal(errorResponse)
	if err != nil {
		logger.Error("Failed to encode error response: %v", err)
		// Fallback to plain text error
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		if _, writeErr := fmt.Fprintf(ctx, "%s: %s", errorType, message); writeErr != nil {
			logger.Error("Failed to write error response: %v", writeErr)
		}
		return
	}

	if writeErr := s.writeCompressedResponse(ctx, responseData); writeErr != nil {
		logger.Error("Failed to write response data: %v", writeErr)
	}
}

// sendDetailedError sends a comprehensive error response with full debugging information.
func (s *Server) sendDetailedError(ctx *fasthttp.RequestCtx, errorType string, err error) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusInternalServerError)

	var errorResponse map[string]any

	// Check if this is a DetailedError with comprehensive debugging info
	if detailedErr, ok := err.(*tsengine.DetailedError); ok {
		// Always log the full detailed error for server-side debugging
		logger.Error("Detailed error occurred:")
		logger.Error("%s", detailedErr.String())

		// Check if debug mode is enabled (you can add this to config later)
		debugMode := s.cfg != nil && s.cfg.Debug

		if debugMode {
			// In debug mode, include full debugging information
			errorResponse = map[string]any{
				"status": "error",
				"error": map[string]any{
					"type":               errorType,
					"message":            detailedErr.Message,
					"timestamp":          detailedErr.Timestamp,
					"source_location":    detailedErr.SourceLocation,
					"error_type":         detailedErr.ErrorType,
					"go_stacktrace":      detailedErr.GoStackTrace,
					"js_stacktrace":      detailedErr.JSStackTrace,
					"context":            detailedErr.Context,
					"runtime_debug_info": detailedErr.RuntimeDebugInfo,
					"debug_mode":         true,
				},
			}
		} else {
			// In production mode, provide limited but useful error information
			errorResponse = map[string]any{
				"status": "error",
				"error": map[string]any{
					"type":            errorType,
					"message":         detailedErr.Message,
					"timestamp":       detailedErr.Timestamp,
					"source_location": detailedErr.SourceLocation,
					"error_type":      detailedErr.ErrorType,
					"debug_mode":      false,
				},
			}

			// Add limited context information (exclude sensitive details)
			if detailedErr.Context != nil {
				limitedContext := make(map[string]any)
				for key, value := range detailedErr.Context {
					// Only include non-sensitive context keys
					switch key {
					case "typescript_file", "timeout_seconds", "execution_time_ms",
						"compilation_time_ms", "query", "params_count", "column_count":
						limitedContext[key] = value
					}
				}
				if len(limitedContext) > 0 {
					errorResponse["error"].(map[string]any)["context"] = limitedContext
				}
			}

			// Add limited stacktrace (just the top few frames, function names only)
			if len(detailedErr.GoStackTrace) > 0 {
				limitedTrace := make([]map[string]any, 0)
				maxFrames := 3 // Only show top 3 frames in production
				for i := 0; i < len(detailedErr.GoStackTrace) && i < maxFrames; i++ {
					frame := detailedErr.GoStackTrace[i]
					limitedTrace = append(limitedTrace, map[string]any{
						"function": frame.Function,
						"package":  frame.Package,
					})
				}
				errorResponse["error"].(map[string]any)["stacktrace_preview"] = limitedTrace
			}
		}
	} else {
		// Fallback to standard error response
		errorResponse = map[string]any{
			"status": "error",
			"error": map[string]any{
				"type":    errorType,
				"message": err.Error(),
			},
		}
	}

	responseData, marshalErr := json.Marshal(errorResponse)
	if marshalErr != nil {
		logger.Error("Failed to encode detailed error response: %v", marshalErr)
		// Fallback to basic error
		s.sendError(ctx, errorType, err.Error())
		return
	}

	if writeErr := s.writeCompressedResponse(ctx, responseData); writeErr != nil {
		logger.Error("Failed to write detailed error response: %v", writeErr)
	}
}

// sendJSONParseError sends a 400 Bad Request response for JSON parsing errors.
func (s *Server) sendJSONParseError(ctx *fasthttp.RequestCtx, err error) {
	ctx.SetStatusCode(fasthttp.StatusBadRequest)
	ctx.SetContentType("application/json")
	errorResponse := map[string]any{
		"status":  "error",
		"message": "Invalid request format",
		"error":   err.Error(),
	}
	if responseData, marshalErr := json.Marshal(errorResponse); marshalErr == nil {
		if writeErr := s.writeCompressedResponse(ctx, responseData); writeErr != nil {
			logger.Error("Failed to write error response: %v", writeErr)
		}
	} else {
		if writeErr := s.writeCompressedResponse(ctx, []byte(`{"status":"error","message":"Invalid request format"}`)); writeErr != nil {
			logger.Error("Failed to write fallback error response: %v", writeErr)
		}
	}
}
