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

// parseRequestBody parses the request body from JSON for POST, PUT, and PATCH requests.
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
