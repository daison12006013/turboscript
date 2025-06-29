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

// Package tsengine provides turboEmail utilities for email sending.
package tsengine

import (
	"fmt"

	"github.com/daison12006013/turboscript/internal/email"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/dop251/goja"
)

// TurboEmailUtils provides shared turboEmail functionality.
type TurboEmailUtils struct {
	emailService *email.Service
}

// NewTurboEmailUtils creates a new turboEmail utilities instance.
func NewTurboEmailUtils(emailService *email.Service) *TurboEmailUtils {
	return &TurboEmailUtils{
		emailService: emailService,
	}
}

// SendEmail sends an email using the configured email service.
// This function is used by both sync and async turboEmail implementations.
// It handles parameter parsing, email validation, and email sending.
//
// Parameters:
//   - call: The goja function call containing email configuration
//   - rt: The goja runtime for error reporting
//
// Returns:
//   - goja.Value: A promise that resolves when the email is sent
func (teu *TurboEmailUtils) SendEmail(call goja.FunctionCall, rt *goja.Runtime) goja.Value {
	logger.Debug("📧 turboEmail called with %d arguments", len(call.Arguments))

	// Validate argument count
	if len(call.Arguments) < 1 {
		panic(rt.NewGoError(fmt.Errorf("turboEmail requires 1 argument: email configuration object")))
	}

	// Extract email configuration
	configValue := call.Arguments[0]
	if goja.IsUndefined(configValue) || goja.IsNull(configValue) {
		panic(rt.NewGoError(fmt.Errorf("email configuration cannot be null or undefined")))
	}

	// Convert goja value to Go map
	exported := configValue.Export()
	configMap, ok := exported.(map[string]any)
	if !ok {
		panic(rt.NewGoError(fmt.Errorf("email configuration must be an object")))
	}

	// Parse email configuration
	emailReq, err := teu.parseEmailConfig(configMap)
	if err != nil {
		panic(rt.NewGoError(fmt.Errorf("invalid email configuration: %w", err)))
	}

	logger.Debug("📝 Sending email: to=%v, subject=%s, driver=%s", emailReq.To, emailReq.Subject, emailReq.Driver)

	// Check if email service is available
	if teu.emailService == nil {
		panic(rt.NewGoError(fmt.Errorf("email service is not initialized")))
	}

	// Send the email
	if err := teu.emailService.SendEmail(emailReq); err != nil {
		panic(rt.NewGoError(fmt.Errorf("failed to send email: %w", err)))
	}

	logger.Debug("✅ Email sent successfully to %v", emailReq.To)

	// Return undefined for emails (they don't return values)
	return goja.Undefined()
}

// parseEmailConfig parses the email configuration from a map.
func (teu *TurboEmailUtils) parseEmailConfig(configMap map[string]any) (*email.Request, error) {
	req := &email.Request{}

	// Parse required fields
	if err := teu.parseRecipients(configMap, req); err != nil {
		return nil, err
	}

	if err := teu.parseRequiredString(configMap, "subject", &req.Subject); err != nil {
		return nil, err
	}

	if err := teu.parseRequiredString(configMap, "content", &req.Content); err != nil {
		return nil, err
	}

	if err := teu.parseRequiredString(configMap, "driver", &req.Driver); err != nil {
		return nil, err
	}

	// Parse optional fields
	teu.parseOptionalString(configMap, "from", &req.From)
	teu.parseOptionalString(configMap, "html", &req.HTML)

	// Parse CC and BCC
	if err := teu.parseOptionalRecipientList(configMap, "cc", &req.CC); err != nil {
		return nil, err
	}

	if err := teu.parseOptionalRecipientList(configMap, "bcc", &req.BCC); err != nil {
		return nil, err
	}

	// Parse attachments
	if err := teu.parseAttachments(configMap, req); err != nil {
		return nil, err
	}

	return req, nil
}

// parseRecipients parses the 'to' field from the config map.
func (teu *TurboEmailUtils) parseRecipients(configMap map[string]any, req *email.Request) error {
	to, exists := configMap["to"]
	if !exists {
		return fmt.Errorf("'to' field is required")
	}

	if toStr, ok := to.(string); ok {
		// Handle single string recipient
		req.To = []string{toStr}
		return nil
	}

	if toArray, ok := to.([]any); ok {
		// Handle array of recipients
		req.To = make([]string, len(toArray))
		for i, v := range toArray {
			if str, ok := v.(string); ok {
				req.To[i] = str
			} else {
				return fmt.Errorf("'to' array must contain only strings")
			}
		}
		return nil
	}

	return fmt.Errorf("'to' must be a string or array of strings")
}

// parseRequiredString parses a required string field from the config map.
func (teu *TurboEmailUtils) parseRequiredString(configMap map[string]any, field string, target *string) error {
	value, exists := configMap[field]
	if !exists {
		return fmt.Errorf("'%s' field is required", field)
	}

	if strValue, ok := value.(string); ok {
		*target = strValue
		return nil
	}

	return fmt.Errorf("'%s' must be a string", field)
}

// parseOptionalString parses an optional string field from the config map.
func (teu *TurboEmailUtils) parseOptionalString(configMap map[string]any, field string, target *string) {
	if value, exists := configMap[field]; exists {
		if strValue, ok := value.(string); ok {
			*target = strValue
		}
	}
}

// parseOptionalRecipientList parses an optional recipient list (cc or bcc) from the config map.
func (teu *TurboEmailUtils) parseOptionalRecipientList(configMap map[string]any, field string, target *[]string) error {
	value, exists := configMap[field]
	if !exists {
		return nil // Field doesn't exist, which is fine for optional fields
	}

	// Handle single string recipient
	if strValue, ok := value.(string); ok {
		*target = []string{strValue}
		return nil
	}

	// Handle array of recipients
	arrayValue, ok := value.([]any)
	if !ok {
		return fmt.Errorf("'%s' must be a string or array of strings", field)
	}

	list := make([]string, len(arrayValue))
	for i, v := range arrayValue {
		str, ok := v.(string)
		if !ok {
			return fmt.Errorf("'%s' array must contain only strings", field)
		}
		list[i] = str
	}
	*target = list
	return nil
}

// parseAttachments parses the attachments from the config map.
func (teu *TurboEmailUtils) parseAttachments(configMap map[string]any, req *email.Request) error {
	attachments, exists := configMap["attachments"]
	if !exists {
		return nil // Attachments are optional
	}

	attachArray, ok := attachments.([]any)
	if !ok {
		return fmt.Errorf("'attachments' must be an array")
	}

	req.Attachments = make([]email.Attachment, len(attachArray))
	for i, v := range attachArray {
		attachMap, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("attachment must be an object")
		}

		// Create a new attachment
		attach := email.Attachment{}

		// Parse required filename field
		filename, exists := attachMap["filename"]
		if !exists {
			return fmt.Errorf("attachment filename is required")
		}

		filenameStr, ok := filename.(string)
		if !ok {
			return fmt.Errorf("attachment filename must be a string")
		}
		attach.Filename = filenameStr

		// Parse required content field
		content, exists := attachMap["content"]
		if !exists {
			return fmt.Errorf("attachment content is required")
		}

		contentStr, ok := content.(string)
		if !ok {
			return fmt.Errorf("attachment content must be a string")
		}
		attach.Content = contentStr

		// Parse optional contentType field
		if contentType, exists := attachMap["contentType"]; exists {
			if contentTypeStr, ok := contentType.(string); ok {
				attach.ContentType = contentTypeStr
			}
		}

		// Add the attachment to the request
		req.Attachments[i] = attach
	}

	return nil
}
