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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/performance"
	"github.com/valyala/fasthttp"
)

// parseRequestToEvent parses the incoming HTTP request into an event object for TypeScript execution.
func (s *Server) parseRequestToEvent(ctx *fasthttp.RequestCtx, ep config.EndpointConfig, perfCtx *performance.RequestContext) (map[string]any, error) {
	if perfCtx != nil {
		perfCtx.StartRequestParsing()
		defer perfCtx.EndRequestParsing()
	}

	pathParams := s.extractPathParams(ep.Route, string(ctx.Path()))
	logger.Debug("Path parameters: %+v", pathParams)

	queryParams := make(map[string]string)
	ctx.QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})
	logger.Debug("Query parameters: %+v", queryParams)

	body, err := s.parseRequestBody(ctx)
	if err != nil {
		return nil, err
	}
	headers := s.parseRequestHeaders(ctx)

	return map[string]any{
		"method":          string(ctx.Method()),
		"path":            string(ctx.Path()),
		"headers":         headers,
		"queryParameters": queryParams,
		"pathParameters":  pathParams,
		"body":            body,
		"env":             s.cfg.GetEnvironmentVariables(),
	}, nil
}

// parseRequestBody parses the request body from JSON for POST, PUT, and PATCH requests,
// or multipart/form-data for file uploads, or raw binary data for direct binary uploads.
func (s *Server) parseRequestBody(ctx *fasthttp.RequestCtx) (map[string]any, error) {
	var body map[string]any
	method := string(ctx.Method())

	if method != "POST" && method != "PUT" && method != "PATCH" {
		return make(map[string]any), nil
	}

	postBody := ctx.PostBody()
	if len(postBody) == 0 {
		return make(map[string]any), nil
	}

	// Check content type to determine parsing method
	contentType := string(ctx.Request.Header.ContentType())

	if strings.Contains(contentType, "multipart/form-data") {
		// Parse multipart/form-data for file uploads
		return s.parseMultipartBody(ctx)
	} else if strings.HasPrefix(contentType, "image/") ||
			  strings.HasPrefix(contentType, "video/") ||
			  strings.HasPrefix(contentType, "audio/") ||
			  contentType == "application/pdf" ||
			  contentType == "application/octet-stream" {
		// Handle raw binary uploads
		return s.parseRawBinaryBody(ctx, contentType)
	}

	// Default: parse as JSON
	if err := json.Unmarshal(postBody, &body); err != nil {
		logger.Debug("Body parsing failed: %v", err)
		return nil, fmt.Errorf("invalid JSON in request body: %w", err)
	}

	logger.Debug("Request body: %+v", body)
	return body, nil
}

// parseRequestHeaders extracts all headers from the HTTP request.
func (s *Server) parseRequestHeaders(ctx *fasthttp.RequestCtx) map[string]string {
	headers := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})
	logger.Debug("Request headers: %+v", headers)
	return headers
}

// parseMultipartBody parses multipart/form-data requests for file uploads.
func (s *Server) parseMultipartBody(ctx *fasthttp.RequestCtx) (map[string]any, error) {
	// Extract boundary from Content-Type header
	contentType := string(ctx.Request.Header.ContentType())
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multipart content type: %w", err)
	}

	boundary, ok := params["boundary"]
	if !ok {
		return nil, fmt.Errorf("missing boundary in multipart content type")
	}

	// Create multipart reader
	reader := multipart.NewReader(strings.NewReader(string(ctx.PostBody())), boundary)

	files := []map[string]any{}
	fields := make(map[string]string)

	// Parse each part
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("Error reading multipart part: %v", err)
			continue
		}

		// Read part data
		partData, err := io.ReadAll(part)
		if err != nil {
			logger.Error("Error reading part data: %v", err)
			part.Close()
			continue
		}
		part.Close()

		if part.FileName() != "" {
			// This is a file part
			mimeType := part.Header.Get("Content-Type")
			if mimeType == "" {
				// Try to detect MIME type from file extension
				mimeType = mime.TypeByExtension(filepath.Ext(part.FileName()))
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}
			}

			file := map[string]any{
				"filename": part.FileName(),
				"data":     base64.StdEncoding.EncodeToString(partData), // Convert to base64 for TypeScript
				"size":     len(partData),
				"mimeType": mimeType,
			}
			files = append(files, file)

			logger.Debug("Parsed file: %s (%d bytes, %s)", part.FileName(), len(partData), mimeType)
		} else {
			// This is a form field
			fieldName := part.FormName()
			if fieldName != "" {
				fields[fieldName] = string(partData)
				logger.Debug("Parsed field: %s = %s", fieldName, string(partData))
			}
		}
	}

	result := map[string]any{
		"files":  files,
		"fields": fields,
	}

	logger.Debug("Parsed multipart body: %d files, %d fields", len(files), len(fields))
	return result, nil
}

// parseRawBinaryBody parses raw binary data uploads.
func (s *Server) parseRawBinaryBody(ctx *fasthttp.RequestCtx, contentType string) (map[string]any, error) {
	postBody := ctx.PostBody()

	// Convert binary data to base64 for TypeScript consumption
	binaryDataB64 := base64.StdEncoding.EncodeToString(postBody)

	result := map[string]any{
		"binaryData": binaryDataB64,
		"contentType": contentType,
		"size": len(postBody),
	}

	logger.Debug("Parsed raw binary body: %d bytes, content-type: %s", len(postBody), contentType)
	return result, nil
}
