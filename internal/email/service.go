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

// Package email provides email sending capabilities for TurboScript.
//
// This package supports multiple email drivers including SMTP, Mailgun, AWS SES,
// SendGrid, and Postmark. The driver is selected based on the configuration
// and can be overridden on a per-message basis.
package email

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
)

// Request represents an email to be sent.
type Request struct {
	To          []string     `json:"to"`
	From        string       `json:"from,omitempty"`
	Subject     string       `json:"subject"`
	Content     string       `json:"content"`
	HTML        string       `json:"html,omitempty"`
	Driver      string       `json:"driver,omitempty"`
	CC          []string     `json:"cc,omitempty"`
	BCC         []string     `json:"bcc,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents an email attachment.
type Attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // base64 encoded
	ContentType string `json:"contentType,omitempty"`
}

// Service provides email sending functionality.
type Service struct {
	config       *config.EmailConfig
	serverConfig *config.ServerConfig // For accessing HTTP timeout and other server settings
}

// NewService creates a new email service instance.
func NewService(cfg *config.EmailConfig) *Service {
	return &Service{
		config: cfg,
	}
}

// NewServiceWithServerConfig creates a new email service instance with server configuration.
func NewServiceWithServerConfig(cfg *config.EmailConfig, serverCfg *config.ServerConfig) *Service {
	return &Service{
		config:       cfg,
		serverConfig: serverCfg,
	}
}

// getHTTPTimeout returns the HTTP timeout from server config, with a default fallback.
func (s *Service) getHTTPTimeout() int {
	if s.serverConfig != nil && s.serverConfig.HTTPTimeout > 0 {
		return s.serverConfig.HTTPTimeout
	}
	return 30 // Default timeout
}

// SendEmail sends an email using the configured driver.
func (s *Service) SendEmail(req *Request) error {
	// Use configured default driver if none specified
	driver := req.Driver
	if driver == "" {
		driver = s.config.DefaultDriver
	}

	// Set default from address if not specified
	if req.From == "" {
		req.From = s.config.FromAddress
	}

	logger.Info("Sending email via %s driver to %v", driver, req.To)

	switch driver {
	case "smtp":
		return s.sendViaSMTP(req)
	case "mailgun":
		return s.sendViaMailgun(req)
	case "ses":
		return s.sendViaSES(req)
	case "sendgrid":
		return s.sendViaSendGrid(req)
	case "postmark":
		return s.sendViaPostmark(req)
	default:
		return fmt.Errorf("unsupported email driver: %s", driver)
	}
}

// sendViaSMTP sends email using SMTP driver.
func (s *Service) sendViaSMTP(req *Request) error {
	smtpConfig := s.config.SMTP

	// Create message
	var msg bytes.Buffer
	msg.WriteString(fmt.Sprintf("From: %s\r\n", req.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(req.To, ", ")))
	if len(req.CC) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(req.CC, ", ")))
	}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", req.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")

	if req.HTML != "" {
		msg.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(req.HTML)
	} else {
		msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(req.Content)
	}

	// Create all recipients list
	allRecipients := make([]string, 0, len(req.To)+len(req.CC)+len(req.BCC))
	allRecipients = append(allRecipients, req.To...)
	allRecipients = append(allRecipients, req.CC...)
	allRecipients = append(allRecipients, req.BCC...)

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)

	if smtpConfig.Encryption == "ssl" {
		return s.sendViaSSL(smtpConfig, addr, req, allRecipients, msg.Bytes())
	}

	return s.sendViaPlainOrTLS(smtpConfig, addr, req, allRecipients, msg.Bytes())
}

// sendViaSSL sends email using SSL connection.
func (s *Service) sendViaSSL(smtpConfig config.SMTPConfig, addr string, req *Request, allRecipients []string, msgBytes []byte) error {
	tlsConfig := &tls.Config{
		ServerName: smtpConfig.Host,
		MinVersion: tls.VersionTLS12, // Minimum TLS 1.2 for security
	}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server with SSL: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			logger.Warn("Failed to close TLS connection: %v", err)
		}
	}()

	client, err := smtp.NewClient(conn, smtpConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer func() {
		if err := client.Quit(); err != nil {
			logger.Warn("Failed to quit SMTP client: %v", err)
		}
	}()

	if err := s.authenticateSMTP(client, smtpConfig); err != nil {
		return err
	}

	return s.sendSMTPMessage(client, req.From, allRecipients, msgBytes)
}

// sendViaPlainOrTLS sends email using plain or STARTTLS connection.
func (s *Service) sendViaPlainOrTLS(smtpConfig config.SMTPConfig, addr string, req *Request, allRecipients []string, msgBytes []byte) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer func() {
		if err := client.Quit(); err != nil {
			logger.Warn("Failed to quit SMTP client: %v", err)
		}
	}()

	if smtpConfig.Encryption == "tls" {
		tlsConfig := &tls.Config{
			ServerName: smtpConfig.Host,
			MinVersion: tls.VersionTLS12, // Minimum TLS 1.2 for security
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	if err := s.authenticateSMTP(client, smtpConfig); err != nil {
		return err
	}

	return s.sendSMTPMessage(client, req.From, allRecipients, msgBytes)
}

// authenticateSMTP performs SMTP authentication if credentials are provided.
func (s *Service) authenticateSMTP(client *smtp.Client, smtpConfig config.SMTPConfig) error {
	if smtpConfig.Username != "" && smtpConfig.Password != "" {
		auth := smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}
	}
	return nil
}

// sendSMTPMessage sends the actual SMTP message.
func (s *Service) sendSMTPMessage(client *smtp.Client, from string, to []string, msg []byte) error {
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer func() {
		if err := writer.Close(); err != nil {
			logger.Warn("Failed to close writer: %v", err)
		}
	}()

	if _, err := writer.Write(msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// sendViaMailgun sends email using Mailgun API.
func (s *Service) sendViaMailgun(req *Request) error {
	mailgunConfig := s.config.Mailgun

	// Mailgun API URL
	baseURL := "https://api.mailgun.net"
	if mailgunConfig.Region == "eu" {
		baseURL = "https://api.eu.mailgun.net"
	}
	url := fmt.Sprintf("%s/v3/%s/messages", baseURL, mailgunConfig.Domain)

	// Prepare form data
	payload := map[string]any{
		"from":    req.From,
		"to":      strings.Join(req.To, ", "),
		"subject": req.Subject,
	}

	if req.HTML != "" {
		payload["html"] = req.HTML
	} else {
		payload["text"] = req.Content
	}

	if len(req.CC) > 0 {
		payload["cc"] = strings.Join(req.CC, ", ")
	}
	if len(req.BCC) > 0 {
		payload["bcc"] = strings.Join(req.BCC, ", ")
	}

	return s.sendHTTPRequest("POST", url, payload, map[string]string{
		"Authorization": "Basic " + encodeBasicAuth("api", mailgunConfig.APIKey),
		"Content-Type":  "application/json",
	})
}

// sendViaSES sends email using AWS SES (simplified implementation).
func (s *Service) sendViaSES(_ *Request) error {
	// Note: This is a simplified implementation
	// For production use, consider using the official AWS SDK
	logger.Warn("SES driver is not fully implemented. Consider using AWS SDK for production.")
	return fmt.Errorf("SES driver not fully implemented")
}

// sendViaSendGrid sends email using SendGrid API.
func (s *Service) sendViaSendGrid(req *Request) error {
	sendGridConfig := s.config.SendGrid

	url := "https://api.sendgrid.com/v3/mail/send"

	// Prepare SendGrid payload
	payload := map[string]any{
		"personalizations": []map[string]any{
			{
				"to": func() []map[string]string {
					var recipients []map[string]string
					for _, email := range req.To {
						recipients = append(recipients, map[string]string{"email": email})
					}
					return recipients
				}(),
			},
		},
		"from": map[string]string{
			"email": req.From,
		},
		"subject": req.Subject,
	}

	if req.HTML != "" {
		payload["content"] = []map[string]string{
			{
				"type":  "text/html",
				"value": req.HTML,
			},
		}
	} else {
		payload["content"] = []map[string]string{
			{
				"type":  "text/plain",
				"value": req.Content,
			},
		}
	}

	return s.sendHTTPRequest("POST", url, payload, map[string]string{
		"Authorization": "Bearer " + sendGridConfig.APIKey,
		"Content-Type":  "application/json",
	})
}

// sendViaPostmark sends email using Postmark API.
func (s *Service) sendViaPostmark(req *Request) error {
	postmarkConfig := s.config.Postmark

	url := "https://api.postmarkapp.com/email"

	// Prepare Postmark payload
	payload := map[string]any{
		"From":    req.From,
		"To":      strings.Join(req.To, ", "),
		"Subject": req.Subject,
	}

	if req.HTML != "" {
		payload["HtmlBody"] = req.HTML
	} else {
		payload["TextBody"] = req.Content
	}

	if len(req.CC) > 0 {
		payload["Cc"] = strings.Join(req.CC, ", ")
	}
	if len(req.BCC) > 0 {
		payload["Bcc"] = strings.Join(req.BCC, ", ")
	}

	return s.sendHTTPRequest("POST", url, payload, map[string]string{
		"X-Postmark-Server-Token": postmarkConfig.ServerToken,
		"Content-Type":            "application/json",
		"Accept":                  "application/json",
	})
}

// sendHTTPRequest sends an HTTP request with JSON payload.
func (s *Service) sendHTTPRequest(method, url string, payload any, headers map[string]string) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{
		Timeout: time.Duration(s.getHTTPTimeout()) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Warn("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email API error (status %d): %s", resp.StatusCode, string(body))
	}

	logger.Info("Email sent successfully")
	return nil
}

// encodeBasicAuth creates a basic auth string.
func encodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
