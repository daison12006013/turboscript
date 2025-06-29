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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// EmailVerificationTestConfig holds configuration for email verification tests.
type EmailVerificationTestConfig struct {
	BaseURL string
}

// VerificationTestResponse represents a typical API response structure.
type VerificationTestResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  []string    `json:"errors,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// UserCreationTestRequest represents the user creation payload for verification tests.
type UserCreationTestRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

// UserCreationTestResponse represents the user creation response.
type UserCreationTestResponse struct {
	UID       string    `json:"uid"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// EmailVerificationData represents the data returned from email verification.
type EmailVerificationData struct {
	Email      string    `json:"email"`
	VerifiedAt time.Time `json:"verified_at"`
	Message    string    `json:"message"`
}

// generateTestEmail creates a unique email address for testing.
func generateTestEmail() string {
	return fmt.Sprintf("verify-test-%d-%d@example.com", time.Now().Unix(), rand.Intn(10000))
}

// generateConfirmationTokenForTest generates an encrypted confirmation token for testing.
func generateConfirmationTokenForTest(uid string) string {
	secret := "turboscript-dev-app-secret"
	expirationTime := time.Now().Add(24*time.Hour).UnixNano() / int64(time.Millisecond)
	payload := fmt.Sprintf("%s:%d", uid, expirationTime)
	return encrypt(payload, secret)
}

// deriveKey generates a cryptographic key from the secret
func deriveKey(secret string) []byte {
	key := make([]byte, 0, 32)
	hash := uint32(5381) // DJB2 hash initial value

	// Create a longer key by processing the secret multiple times
	for round := 0; round < 4; round++ {
		for _, char := range secret {
			hash = ((hash << 5) + hash) + uint32(char)
			hash = hash & 0xFFFFFFFF // Keep it 32-bit
		}
		key = append(key, byte(hash&0xFF))
	}

	// Expand key to 32 bytes
	for len(key) < 32 {
		prevIndex := len(key) - 1
		newByte := (key[prevIndex] ^ key[prevIndex%8]) & 0xFF
		key = append(key, newByte)
	}

	return key
}

// generateIV generates a random IV (Initialization Vector)
func generateIV() []byte {
	iv := make([]byte, 16)
	timestamp := time.Now().UnixNano()
	random := rand.Int63n(1000000)

	// Use timestamp and random for IV generation
	seed := timestamp + random
	value := seed

	for i := 0; i < 16; i++ {
		value = (value*1103515245 + 12345) & 0xFFFFFFFF
		iv[i] = byte(value & 0xFF)
	}

	return iv
}

// encryptData encrypts data using AES-like algorithm
func encryptData(data string, key []byte, iv []byte) []byte {
	dataBytes := []byte(data)
	encrypted := make([]byte, len(dataBytes))

	for i, dataByte := range dataBytes {
		keyByte := key[i%len(key)]
		ivByte := iv[i%len(iv)]

		// XOR encryption with key and IV
		encryptedByte := dataByte ^ keyByte ^ ivByte

		// Add some additional obfuscation
		encryptedByte = (encryptedByte + byte(i) + keyByte) & 0xFF

		encrypted[i] = encryptedByte
	}

	return encrypted
}

// encodeBase64Url encodes bytes to base64url format
func encodeBase64Url(data []byte) string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := ""

	for i := 0; i < len(data); i += 3 {
		a := int(data[i])
		b := 0
		c := 0

		if i+1 < len(data) {
			b = int(data[i+1])
		}
		if i+2 < len(data) {
			c = int(data[i+2])
		}

		bitmap := (a << 16) | (b << 8) | c

		result += string(chars[(bitmap>>18)&63])
		result += string(chars[(bitmap>>12)&63])
		result += string(chars[(bitmap>>6)&63])
		result += string(chars[bitmap&63])
	}

	// Remove padding equivalent
	for len(result)%4 != 0 && strings.HasSuffix(result, "A") {
		result = result[:len(result)-1]
	}

	return result
}

// encrypt encrypts a payload to create a secure token
func encrypt(payload, secret string) string {
	key := deriveKey(secret)
	iv := generateIV()
	encrypted := encryptData(payload, key, iv)

	// Combine IV and encrypted data
	combined := append(iv, encrypted...)

	return encodeBase64Url(combined)
}

// generateExpiredConfirmationToken generates an expired encrypted confirmation token for testing.
func generateExpiredConfirmationToken(uid string) string {
	secret := "turboscript-dev-app-secret"
	// Create a token that's expired (1 hour ago)
	expirationTime := time.Now().Add(-1*time.Hour).UnixNano() / int64(time.Millisecond)
	payload := fmt.Sprintf("%s:%d", uid, expirationTime)
	return encrypt(payload, secret)
}

// TestEmailVerificationFlow performs comprehensive email verification testing.
// This test requires a running server and database.
func TestEmailVerificationFlow(t *testing.T) {
	// Skip if not in E2E test mode
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("Skipping email verification E2E test. Set E2E_TEST=true to run.")
	}

	baseURL := os.Getenv("TEST_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:7890"
	}

	config := EmailVerificationTestConfig{
		BaseURL: baseURL,
	}

	t.Run("Complete Email Verification Flow", func(t *testing.T) {
		testCompleteEmailVerificationFlow(t, config)
	})

	t.Run("Email Verification Validation", func(t *testing.T) {
		testEmailVerificationValidation(t, config)
	})

	t.Run("Email Verification Edge Cases", func(t *testing.T) {
		testEmailVerificationEdgeCases(t, config)
	})
}

// testCompleteEmailVerificationFlow tests the complete user creation and email verification flow.
func testCompleteEmailVerificationFlow(t *testing.T, config EmailVerificationTestConfig) {
	// Step 1: Create a new user
	userEmail := generateTestEmail()
	userPayload := UserCreationTestRequest{
		Name:            "Verification Test User",
		Email:           userEmail,
		Password:        "TestPassword123!",
		ConfirmPassword: "TestPassword123!",
	}

	payloadBytes, err := json.Marshal(userPayload)
	if err != nil {
		t.Fatalf("Failed to marshal user creation payload: %v", err)
	}

	// Create user
	resp, err := http.Post(
		fmt.Sprintf("%s/users", config.BaseURL),
		"application/json",
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	var createResponse VerificationTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
		t.Fatalf("Failed to decode user creation response: %v", err)
	}

	if createResponse.Status != "success" {
		t.Fatalf("Expected success status, got %s", createResponse.Status)
	}

	// Extract user data
	userData, ok := createResponse.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected user data to be a map")
	}

	userUID, ok := userData["uid"].(string)
	if !ok {
		t.Fatalf("Expected user UID to be a string")
	}

	// Step 2: Generate confirmation token and verify email
	confirmationToken := generateConfirmationTokenForTest(userUID)

	verifyResp, err := http.Get(
		fmt.Sprintf("%s/users/verify/%s", config.BaseURL, confirmationToken),
	)
	if err != nil {
		t.Fatalf("Failed to verify email: %v", err)
	}
	defer verifyResp.Body.Close()

	if verifyResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for email verification, got %d", verifyResp.StatusCode)
	}

	var verifyResponse VerificationTestResponse
	if err := json.NewDecoder(verifyResp.Body).Decode(&verifyResponse); err != nil {
		t.Fatalf("Failed to decode email verification response: %v", err)
	}

	if verifyResponse.Status != "success" {
		t.Fatalf("Expected success status for email verification, got %s", verifyResponse.Status)
	}

	if verifyResponse.Message != "Email verified successfully" {
		t.Errorf("Expected 'Email verified successfully' message, got %s", verifyResponse.Message)
	}

	// Verify the response data contains email and verified_at
	verifyData, ok := verifyResponse.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected verification data to be a map")
	}

	if verifyData["email"] != userEmail {
		t.Errorf("Expected email %s, got %s", userEmail, verifyData["email"])
	}

	if verifyData["verified_at"] == nil {
		t.Error("Expected verified_at to be present and non-nil")
	}

	// Step 3: Try to verify again (should still succeed but indicate already verified)
	secondVerifyResp, err := http.Get(
		fmt.Sprintf("%s/users/verify/%s", config.BaseURL, confirmationToken),
	)
	if err != nil {
		t.Fatalf("Failed to verify email second time: %v", err)
	}
	defer secondVerifyResp.Body.Close()

	if secondVerifyResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 for second email verification, got %d", secondVerifyResp.StatusCode)
	}

	var secondVerifyResponse VerificationTestResponse
	if err := json.NewDecoder(secondVerifyResp.Body).Decode(&secondVerifyResponse); err != nil {
		t.Fatalf("Failed to decode second email verification response: %v", err)
	}

	if secondVerifyResponse.Status != "success" {
		t.Fatalf("Expected success status for second email verification, got %s", secondVerifyResponse.Status)
	}

	if secondVerifyResponse.Message != "Email already verified" {
		t.Errorf("Expected 'Email already verified' message, got %s", secondVerifyResponse.Message)
	}
}

// testEmailVerificationValidation tests various validation scenarios.
func testEmailVerificationValidation(t *testing.T, config EmailVerificationTestConfig) {
	testCases := []struct {
		name           string
		token          string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing confirmation token",
			token:          "",
			expectedStatus: http.StatusNotFound, // Route won't match
		},
		{
			name:           "Invalid token format - too short",
			token:          "invalid",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "Invalid or expired confirmation token",
		},
		{
			name:           "Invalid token format - malformed payload",
			token:          "invalidpayload.signature",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "Invalid or expired confirmation token",
		},
		{
			name:           "Invalid token format - missing signature",
			token:          "00000000-0000-0000-0000-000000000000:9999999999999",
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "Invalid or expired confirmation token",
		},
		{
			name:           "Non-existent user with invalid signature",
			token:          fmt.Sprintf("00000000-0000-0000-0000-000000000000:%d.invalidsig", time.Now().Add(24*time.Hour).UnixNano()/int64(time.Millisecond)),
			expectedStatus: http.StatusUnprocessableEntity,
			expectedError:  "Invalid or expired confirmation token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var requestURL string
			if tc.token == "" {
				requestURL = fmt.Sprintf("%s/users/verify/", config.BaseURL)
			} else {
				requestURL = fmt.Sprintf("%s/users/verify/%s", config.BaseURL, tc.token)
			}

			resp, err := http.Get(requestURL)
			if err != nil {
				t.Fatalf("Failed to make verification request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			if tc.expectedError != "" {
				var response VerificationTestResponse
				if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}

				if response.Status != "error" {
					t.Errorf("Expected error status, got %s", response.Status)
				}

				if !strings.Contains(response.Message, tc.expectedError) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tc.expectedError, response.Message)
				}
			}
		})
	}
}

// testEmailVerificationEdgeCases tests edge cases and security scenarios.
func testEmailVerificationEdgeCases(t *testing.T, config EmailVerificationTestConfig) {
	t.Run("Expired token", func(t *testing.T) {
		// Create a user first
		userEmail := generateTestEmail()
		userPayload := UserCreationTestRequest{
			Name:            "Expired Token Test User",
			Email:           userEmail,
			Password:        "TestPassword123!",
			ConfirmPassword: "TestPassword123!",
		}

		payloadBytes, err := json.Marshal(userPayload)
		if err != nil {
			t.Fatalf("Failed to marshal user creation payload: %v", err)
		}

		resp, err := http.Post(
			fmt.Sprintf("%s/users", config.BaseURL),
			"application/json",
			bytes.NewBuffer(payloadBytes),
		)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", resp.StatusCode)
		}

		var createResponse VerificationTestResponse
		if err := json.NewDecoder(resp.Body).Decode(&createResponse); err != nil {
			t.Fatalf("Failed to decode user creation response: %v", err)
		}

		userData, ok := createResponse.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected user data to be a map")
		}

		userUID, ok := userData["uid"].(string)
		if !ok {
			t.Fatalf("Expected user UID to be a string")
		}

		// Generate expired token
		expiredToken := generateExpiredConfirmationToken(userUID)

		verifyResp, err := http.Get(
			fmt.Sprintf("%s/users/verify/%s", config.BaseURL, expiredToken),
		)
		if err != nil {
			t.Fatalf("Failed to verify expired token: %v", err)
		}
		defer verifyResp.Body.Close()

		if verifyResp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("Expected status 422 for expired token, got %d", verifyResp.StatusCode)
		}

		var verifyResponse VerificationTestResponse
		if err := json.NewDecoder(verifyResp.Body).Decode(&verifyResponse); err != nil {
			t.Fatalf("Failed to decode expired token response: %v", err)
		}

		if verifyResponse.Status != "error" {
			t.Errorf("Expected error status for expired token, got %s", verifyResponse.Status)
		}

		if verifyResponse.Message != "Invalid or expired confirmation token" {
			t.Errorf("Expected 'Invalid or expired confirmation token' message, got %s", verifyResponse.Message)
		}
	})

	t.Run("Malformed UUID in token", func(t *testing.T) {
		malformedToken := "invalid-uuid:9999999999999.invalidsig"

		verifyResp, err := http.Get(
			fmt.Sprintf("%s/users/verify/%s", config.BaseURL, malformedToken),
		)
		if err != nil {
			t.Fatalf("Failed to verify malformed token: %v", err)
		}
		defer verifyResp.Body.Close()

		// Should return 422 since the token signature will be invalid
		if verifyResp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("Expected status 422 for malformed UUID, got %d", verifyResp.StatusCode)
		}
	})
}
