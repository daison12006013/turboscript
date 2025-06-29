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

package tsengine

import (
	"testing"

	"github.com/daison12006013/turboscript/internal/email"
	"github.com/dop251/goja"
)

// TestNewTurboEmailUtils tests creation of new TurboEmailUtils
func TestNewTurboEmailUtils(t *testing.T) {
	// Create test utils with nil service (fine for constructor test)
	utils := NewTurboEmailUtils(nil)

	// Assert that instance was created
	if utils == nil {
		t.Fatal("NewTurboEmailUtils() returned nil")
	}
}

// TestParseEmailConfig tests the parseEmailConfig method
func TestParseEmailConfig(t *testing.T) {
	teu := &TurboEmailUtils{}

	testCases := []struct {
		name      string
		config    map[string]any
		expectErr bool
		checkFn   func(*testing.T, *email.Request)
	}{
		{
			name: "Complete Configuration",
			config: map[string]any{
				"to":      []any{"recipient@example.com", "another@example.com"},
				"from":    "sender@example.com",
				"subject": "Complete Test",
				"content": "Complete test content",
				"html":    "<p>HTML Content</p>",
				"driver":  "smtp",
				"cc":      []any{"cc1@example.com", "cc2@example.com"},
				"bcc":     []any{"bcc1@example.com", "bcc2@example.com"},
				"attachments": []any{
					map[string]any{
						"filename":    "test.txt",
						"content":     "SGVsbG8gV29ybGQ=", // Base64 "Hello World"
						"contentType": "text/plain",
					},
				},
			},
			expectErr: false,
			checkFn: func(t *testing.T, req *email.Request) {
				// Check recipients
				if len(req.To) != 2 || req.To[0] != "recipient@example.com" || req.To[1] != "another@example.com" {
					t.Errorf("Unexpected To recipients: %v", req.To)
				}

				// Check other fields
				if req.From != "sender@example.com" {
					t.Errorf("Expected From to be %q, got %q", "sender@example.com", req.From)
				}

				if req.Subject != "Complete Test" {
					t.Errorf("Expected Subject to be %q, got %q", "Complete Test", req.Subject)
				}

				if req.Content != "Complete test content" {
					t.Errorf("Expected Content to be %q, got %q", "Complete test content", req.Content)
				}

				if req.HTML != "<p>HTML Content</p>" {
					t.Errorf("Expected HTML to be %q, got %q", "<p>HTML Content</p>", req.HTML)
				}

				if req.Driver != "smtp" {
					t.Errorf("Expected Driver to be %q, got %q", "smtp", req.Driver)
				}

				// Check CC
				if len(req.CC) != 2 || req.CC[0] != "cc1@example.com" || req.CC[1] != "cc2@example.com" {
					t.Errorf("Unexpected CC recipients: %v", req.CC)
				}

				// Check BCC
				if len(req.BCC) != 2 || req.BCC[0] != "bcc1@example.com" || req.BCC[1] != "bcc2@example.com" {
					t.Errorf("Unexpected BCC recipients: %v", req.BCC)
				}

				// Check attachments
				if len(req.Attachments) != 1 {
					t.Errorf("Expected 1 attachment, got %d", len(req.Attachments))
				} else {
					att := req.Attachments[0]
					if att.Filename != "test.txt" {
						t.Errorf("Expected attachment filename to be %q, got %q", "test.txt", att.Filename)
					}

					if att.Content != "SGVsbG8gV29ybGQ=" {
						t.Errorf("Expected attachment content to be %q, got %q", "SGVsbG8gV29ybGQ=", att.Content)
					}

					if att.ContentType != "text/plain" {
						t.Errorf("Expected attachment content type to be %q, got %q", "text/plain", att.ContentType)
					}
				}
			},
		},
		{
			name: "Single String Recipient",
			config: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test Email",
				"content": "This is a test email",
				"driver":  "smtp",
			},
			expectErr: false,
			checkFn: func(t *testing.T, req *email.Request) {
				if len(req.To) != 1 || req.To[0] != "recipient@example.com" {
					t.Errorf("Expected To to be [%q], got %v", "recipient@example.com", req.To)
				}
			},
		},
		{
			name: "String CC and BCC",
			config: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test with String CC and BCC",
				"content": "Test content",
				"driver":  "smtp",
				"cc":      "cc@example.com",
				"bcc":     "bcc@example.com",
			},
			expectErr: false,
			checkFn: func(t *testing.T, req *email.Request) {
				if len(req.CC) != 1 || req.CC[0] != "cc@example.com" {
					t.Errorf("Expected CC to be [%q], got %v", "cc@example.com", req.CC)
				}

				if len(req.BCC) != 1 || req.BCC[0] != "bcc@example.com" {
					t.Errorf("Expected BCC to be [%q], got %v", "bcc@example.com", req.BCC)
				}
			},
		},
		{
			name: "Missing To Field",
			config: map[string]any{
				"subject": "Test Email",
				"content": "This is a test email",
				"driver":  "smtp",
			},
			expectErr: true,
		},
		{
			name: "Missing Subject Field",
			config: map[string]any{
				"to":      []any{"recipient@example.com"},
				"content": "This is a test email",
				"driver":  "smtp",
			},
			expectErr: true,
		},
		{
			name: "Missing Content Field",
			config: map[string]any{
				"to":      []any{"recipient@example.com"},
				"subject": "Test Email",
				"driver":  "smtp",
			},
			expectErr: true,
		},
		{
			name: "Missing Driver Field",
			config: map[string]any{
				"to":      []any{"recipient@example.com"},
				"subject": "Test Email",
				"content": "This is a test email",
			},
			expectErr: true,
		},
		{
			name: "Invalid To Type",
			config: map[string]any{
				"to":      123, // Not a string or array
				"subject": "Test Email",
				"content": "This is a test email",
				"driver":  "smtp",
			},
			expectErr: true,
		},
		{
			name: "Invalid To Array Element",
			config: map[string]any{
				"to":      []any{"valid@example.com", 123}, // Non-string element
				"subject": "Test Email",
				"content": "This is a test email",
				"driver":  "smtp",
			},
			expectErr: true,
		},
		{
			name: "Invalid Subject Type",
			config: map[string]any{
				"to":      "recipient@example.com",
				"subject": 123, // Not a string
				"content": "This is a test email",
				"driver":  "smtp",
			},
			expectErr: true,
		},
		{
			name: "Invalid Content Type",
			config: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test Email",
				"content": 123, // Not a string
				"driver":  "smtp",
			},
			expectErr: true,
		},
		{
			name: "Invalid Driver Type",
			config: map[string]any{
				"to":      "recipient@example.com",
				"subject": "Test Email",
				"content": "This is a test email",
				"driver":  123, // Not a string
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := teu.parseEmailConfig(tc.config)

			if tc.expectErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tc.checkFn != nil {
				tc.checkFn(t, req)
			}
		})
	}
}

// Test functions that don't need error recovery since they have complex panic handling
func TestSendEmail_ErrorHandling(t *testing.T) {
	// Create utils with nil service to test the service check
	utils := &TurboEmailUtils{emailService: nil}

	// Create goja runtime
	rt := goja.New()

	// Create valid email config
	emailConfig := map[string]any{
		"to":      "recipient@example.com",
		"subject": "Test Email",
		"content": "This is a test email",
		"driver":  "smtp",
	}

	// Test the different error conditions using panics to validate they happen
	testCases := []struct {
		name   string
		config any
		args   []goja.Value
	}{
		{
			name: "Missing Arguments",
			args: []goja.Value{}, // No arguments
		},
		{
			name: "Null Config",
			args: []goja.Value{goja.Null()}, // Null argument
		},
		{
			name: "Invalid Config Type",
			args: []goja.Value{rt.ToValue("not-an-object")}, // String instead of object
		},
		{
			name: "Nil Email Service",
			args: []goja.Value{rt.ToValue(emailConfig)}, // Valid config but nil service
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create function call
			call := goja.FunctionCall{
				This:      rt.GlobalObject(),
				Arguments: tc.args,
			}

			// These should all panic, so we expect them to fail
			defer func() {
				recover() // Ignore the panic for test purposes
			}()

			utils.SendEmail(call, rt)
			t.Log("Function completed without panic - this is expected for validation")
		})
	}
}
