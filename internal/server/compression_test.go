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
	"strings"
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/valyala/fasthttp"
)

func TestCompressionShouldCompress(t *testing.T) {
	tests := []struct {
		name          string
		compression   config.CompressionConfig
		contentType   string
		contentLength int
		acceptHeader  string
		expected      bool
	}{
		{
			name: "Compression enabled, JSON content, large size, gzip accepted",
			compression: config.CompressionConfig{
				Enabled: true,
				MinSize: 1024,
				Level:   6,
			},
			contentType:   "application/json",
			contentLength: 2048,
			acceptHeader:  "gzip, deflate",
			expected:      true,
		},
		{
			name: "Compression disabled",
			compression: config.CompressionConfig{
				Enabled: false,
				MinSize: 1024,
				Level:   6,
			},
			contentType:   "application/json",
			contentLength: 2048,
			acceptHeader:  "gzip, deflate",
			expected:      false,
		},
		{
			name: "Size too small",
			compression: config.CompressionConfig{
				Enabled: true,
				MinSize: 1024,
				Level:   6,
			},
			contentType:   "application/json",
			contentLength: 512,
			acceptHeader:  "gzip, deflate",
			expected:      false,
		},
		{
			name: "Client doesn't accept gzip",
			compression: config.CompressionConfig{
				Enabled: true,
				MinSize: 1024,
				Level:   6,
			},
			contentType:   "application/json",
			contentLength: 2048,
			acceptHeader:  "deflate",
			expected:      false,
		},
		{
			name: "Binary content type",
			compression: config.CompressionConfig{
				Enabled: true,
				MinSize: 1024,
				Level:   6,
			},
			contentType:   "image/jpeg",
			contentLength: 2048,
			acceptHeader:  "gzip, deflate",
			expected:      false,
		},
		{
			name: "HTML content type",
			compression: config.CompressionConfig{
				Enabled: true,
				MinSize: 1024,
				Level:   6,
			},
			contentType:   "text/html; charset=utf-8",
			contentLength: 2048,
			acceptHeader:  "gzip, deflate",
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create server with test configuration
			cfg := &config.Config{
				Compression: tt.compression,
			}
			server := &Server{
				cfg: cfg,
			}

			// Create a fasthttp context with test data
			ctx := &fasthttp.RequestCtx{}
			ctx.Request.Header.Set("Accept-Encoding", tt.acceptHeader)
			ctx.Response.Header.SetContentType(tt.contentType)

			// Test shouldCompress
			result := server.shouldCompress(ctx, tt.contentType, tt.contentLength)

			if result != tt.expected {
				t.Errorf("shouldCompress() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCompressionActualCompression(t *testing.T) {
	// Create server with compression enabled
	cfg := &config.Config{
		Compression: config.CompressionConfig{
			Enabled: true,
			MinSize: 100, // Small size for testing
			Level:   6,
		},
	}
	server := &Server{
		cfg: cfg,
	}

	// Create a large JSON response that should compress well
	largeJSON := `{"message":"` + strings.Repeat("This is a test message that should compress well. ", 100) + `"}`

	// Create fasthttp context
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("Accept-Encoding", "gzip")
	ctx.Response.Header.SetContentType("application/json")

	// Test writeCompressedResponse
	err := server.writeCompressedResponse(ctx, []byte(largeJSON))
	if err != nil {
		t.Fatalf("writeCompressedResponse failed: %v", err)
	}

	// Check that compression headers are set
	contentEncoding := string(ctx.Response.Header.Peek("Content-Encoding"))
	if contentEncoding != "gzip" {
		t.Errorf("Expected Content-Encoding: gzip, got: %s", contentEncoding)
	}

	vary := string(ctx.Response.Header.Peek("Vary"))
	if vary != "Accept-Encoding" {
		t.Errorf("Expected Vary: Accept-Encoding, got: %s", vary)
	}

	// Check that response is smaller than original
	responseBody := ctx.Response.Body()
	if len(responseBody) >= len(largeJSON) {
		t.Errorf("Compressed response (%d bytes) is not smaller than original (%d bytes)",
			len(responseBody), len(largeJSON))
	}

	t.Logf("Original size: %d bytes, Compressed size: %d bytes, Reduction: %.1f%%",
		len(largeJSON), len(responseBody),
		float64(len(largeJSON)-len(responseBody))/float64(len(largeJSON))*100)
}
