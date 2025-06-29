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
	"strings"

	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/performance"
	"github.com/daison12006013/turboscript/internal/templating"
	"github.com/valyala/fasthttp"
)

// unwrapResponseWithType extracts the "response" field from a JSON wrapper
// while preserving the original key ordering, handling cookies, and detecting response type.
func (s *Server) unwrapResponseWithType(wrappedJSON []byte) ([]byte, int, []string, string, error) {
	return unwrapResponseWithType(wrappedJSON)
}

// setContentTypeFromResponseType sets the appropriate content type based on response type.
func (s *Server) setContentTypeFromResponseType(ctx *fasthttp.RequestCtx, responseType string) {
	switch responseType {
	case "html":
		ctx.SetContentType("text/html; charset=utf-8")
	case "templated-html":
		ctx.SetContentType("text/html; charset=utf-8")
	case "markdown":
		ctx.SetContentType("text/markdown; charset=utf-8")
	case responseTypeMarkdownHTML:
		ctx.SetContentType("text/html; charset=utf-8")
	case "text":
		ctx.SetContentType("text/plain; charset=utf-8")
	default: // json or any other type
		ctx.SetContentType("application/json")
	}

	// Set cache headers based on debug mode
	if s.cfg.Debug {
		// Development mode: disable caching for API responses
		ctx.Response.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
		ctx.Response.Header.Set("Pragma", "no-cache")
		ctx.Response.Header.Set("Expires", "0")
	} else {
		// Production mode: allow short-term caching for API responses
		ctx.Response.Header.Set("Cache-Control", "public, max-age=60") // 1 minute
	}
}

// shouldCompress determines if the response should be compressed based on content type and size.
func (s *Server) shouldCompress(ctx *fasthttp.RequestCtx, contentType string, contentLength int) bool {
	// Check if compression is enabled in configuration
	if !s.cfg.Compression.Enabled {
		return false
	}

	// Check if client accepts gzip encoding
	acceptEncoding := string(ctx.Request.Header.Peek("Accept-Encoding"))
	if !strings.Contains(strings.ToLower(acceptEncoding), "gzip") {
		return false
	}

	// Check minimum size threshold
	minSize := s.cfg.Compression.MinSize
	if minSize == 0 {
		minSize = 1024 // default 1KB
	}
	if contentLength < minSize {
		return false
	}

	// Only compress text-based content types
	switch {
	case strings.HasPrefix(contentType, "application/json"):
		return true
	case strings.HasPrefix(contentType, "text/"):
		return true
	case strings.HasPrefix(contentType, "application/javascript"):
		return true
	case strings.HasPrefix(contentType, "application/xml"):
		return true
	default:
		return false
	}
}

// compressResponse compresses the response data using gzip.
func (s *Server) compressResponse(data []byte) []byte {
	var compressed []byte

	// Use fasthttp's builtin compression for better performance
	compressed = fasthttp.AppendGzipBytes(compressed, data)

	return compressed
}

// writeCompressedResponse writes a compressed response to the client.
func (s *Server) writeCompressedResponse(ctx *fasthttp.RequestCtx, data []byte) error {
	// Check if we should compress based on content type and size
	contentType := string(ctx.Response.Header.ContentType())

	// Check if compression is enabled and data meets criteria
	if !s.shouldCompress(ctx, contentType, len(data)) {
		// Write uncompressed
		_, err := ctx.Write(data)
		return err
	}

	// Compress the response
	compressed := s.compressResponse(data)

	// Only use compression if it actually reduces size
	if len(compressed) >= len(data) {
		// Compression didn't help, send uncompressed
		_, writeErr := ctx.Write(data)
		return writeErr
	}

	// Set compression headers
	ctx.Response.Header.Set("Content-Encoding", "gzip")
	ctx.Response.Header.Set("Vary", "Accept-Encoding")

	// Write compressed data
	_, writeErr := ctx.Write(compressed)
	if writeErr != nil {
		return writeErr
	}

	logger.Debug("Response compressed: %d bytes -> %d bytes (%.1f%% reduction)",
		len(data), len(compressed),
		float64(len(data)-len(compressed))/float64(len(data))*100)

	return nil
}

// handleFinalResponse processes the final response and sends it to the client.
func (s *Server) handleFinalResponse(ctx *fasthttp.RequestCtx, rawResponseResult json.RawMessage, perfCtx *performance.RequestContext) {
	if rawResponseResult != nil {
		s.handleNonNilResponse(ctx, rawResponseResult, perfCtx)
	} else {
		s.handleNilResponse(ctx)
	}

	if perfCtx != nil {
		perfCtx.EndResponseProcessing()
	}
}

// handleNonNilResponse processes non-nil responses, checking for wrapper format.
func (s *Server) handleNonNilResponse(ctx *fasthttp.RequestCtx, rawResponseResult json.RawMessage, perfCtx *performance.RequestContext) {
	// Parse the response to check for code and response wrapper
	var responseWrapper map[string]any
	if err := json.Unmarshal(rawResponseResult, &responseWrapper); err == nil {
		if s.hasResponseWrapper(responseWrapper) {
			s.handleWrappedResponse(ctx, rawResponseResult, perfCtx)
			return
		}
	}

	// Default to JSON for direct responses
	ctx.SetContentType("application/json")
	// Direct response without unwrapping
	s.writeDirectResponse(ctx, rawResponseResult)
}

// hasResponseWrapper checks if the response has the expected wrapper structure.
func (s *Server) hasResponseWrapper(responseWrapper map[string]any) bool {
	_, hasCode := responseWrapper["code"]
	_, hasResponse := responseWrapper["response"]
	return hasCode && hasResponse
}

// handleWrappedResponse processes wrapped responses with code, cookies, and type.
func (s *Server) handleWrappedResponse(ctx *fasthttp.RequestCtx, rawResponseResult json.RawMessage, perfCtx *performance.RequestContext) {
	unwrappedResponse, statusCode, cookies, responseType, err := s.unwrapResponseWithType(rawResponseResult)
	if err != nil {
		logger.Error("Failed to unwrap response: %v", err)
		if perfCtx != nil {
			perfCtx.EndResponseProcessing()
		}
		s.sendError(ctx, "Response Unwrapping Error", err.Error())
		return
	}

	s.setCookies(ctx, cookies)
	s.setContentTypeFromResponseType(ctx, responseType)
	s.writeUnwrappedResponse(ctx, unwrappedResponse, statusCode, responseType)
}

// setCookies sets cookies from the response.
func (s *Server) setCookies(ctx *fasthttp.RequestCtx, cookies []string) {
	for _, cookie := range cookies {
		ctx.Response.Header.Add("Set-Cookie", cookie)
		logger.Debug("Set cookie: %s", cookie)
	}
}

// writeUnwrappedResponse writes the unwrapped response with appropriate formatting.
func (s *Server) writeUnwrappedResponse(ctx *fasthttp.RequestCtx, unwrappedResponse []byte, statusCode int, responseType string) {
	ctx.SetStatusCode(statusCode)

	// Handle templated HTML responses (preserve formatting from templating engine)
	if responseType == "templated-html" {
		// For templated HTML responses, preserve all formatting including code blocks
		var htmlContent string
		if err := json.Unmarshal(unwrappedResponse, &htmlContent); err == nil {
			// Write the templated HTML directly without cleaning to preserve formatting
			if writeErr := s.writeCompressedResponse(ctx, []byte(htmlContent)); writeErr != nil {
				logger.Error("Failed to write templated HTML response: %v", writeErr)
			}
			logger.Debug("Sending templated HTML response (preserving formatting)")
			return
		}
		logger.Warn("Failed to parse templated HTML content, sending as-is")
	}

	// Handle HTML responses (clean whitespace for simple HTML)
	if responseType == "html" {
		// For HTML responses, we need to parse the JSON string content and clean it
		var htmlContent string
		if err := json.Unmarshal(unwrappedResponse, &htmlContent); err == nil {
			cleanedHTML := cleanHTMLResponse(htmlContent)
			// Write the cleaned HTML directly (not JSON-encoded)
			if writeErr := s.writeCompressedResponse(ctx, []byte(cleanedHTML)); writeErr != nil {
				logger.Error("Failed to write cleaned HTML response: %v", writeErr)
			}
			logger.Debug("Sending cleaned HTML response")
			return
		}
		logger.Warn("Failed to parse HTML content for cleaning, sending as-is")
	}

	// Handle markdown-html responses (convert markdown to HTML)
	if responseType == responseTypeMarkdownHTML {
		// For markdown-html responses, convert markdown to HTML
		var markdownContent string
		if err := json.Unmarshal(unwrappedResponse, &markdownContent); err == nil {
			htmlContent := templating.ConvertMarkdownToHTML(markdownContent)
			// Don't clean markdown-converted HTML to preserve code block formatting
			// Write the converted HTML directly
			if writeErr := s.writeCompressedResponse(ctx, []byte(htmlContent)); writeErr != nil {
				logger.Error("Failed to write markdown-html response: %v", writeErr)
			}
			logger.Debug("Sending markdown-html response (converted to HTML, preserving formatting)")
			return
		}
		logger.Warn("Failed to parse markdown content for conversion, sending as-is")
	}

	// For other response types or if processing failed, write as-is with compression
	if writeErr := s.writeCompressedResponse(ctx, unwrappedResponse); writeErr != nil {
		logger.Error("Failed to write unwrapped response: %v", writeErr)
	}
	logger.Debug("Sending unwrapped response with preserved key ordering")
}

// writeDirectResponse writes a direct response without unwrapping.
func (s *Server) writeDirectResponse(ctx *fasthttp.RequestCtx, rawResponseResult json.RawMessage) {
	if writeErr := s.writeCompressedResponse(ctx, rawResponseResult); writeErr != nil {
		logger.Error("Failed to write response: %v", writeErr)
	}
	logger.Debug("Sending direct response")
}

// handleNilResponse handles nil responses by sending a default success response.
func (s *Server) handleNilResponse(ctx *fasthttp.RequestCtx) {
	ctx.SetContentType("application/json")
	if writeErr := s.writeCompressedResponse(ctx, []byte(`{"status":"success"}`)); writeErr != nil {
		logger.Error("Failed to write success response: %v", writeErr)
	}
	logger.Debug("Sending default success response")
}
