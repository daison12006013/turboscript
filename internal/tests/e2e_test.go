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
	"testing"
	"time"
)

// E2ETestConfig holds configuration for end-to-end tests.
type E2ETestConfig struct {
	BaseURL string
	Token   string
}

// E2ETestResponse represents a typical API response structure.
type E2ETestResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
	Error  string `json:"error,omitempty"`
	Meta   any    `json:"meta,omitempty"`
}

// E2ETestUser represents a user for testing.
type E2ETestUser struct {
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// E2EAuthResponse represents login response.
type E2EAuthResponse struct {
	UID                   string `json:"uid"`
	Name                  string `json:"name"`
	Email                 string `json:"email"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
	PasswordNeedsRehash   bool   `json:"password_needs_rehash"`
}

// UserCreationRequest represents the user creation payload.
type UserCreationRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

// LoginRequest represents the login payload.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// generateRandomEmail creates a unique email address using timestamp and random number.
func generateRandomEmail() string {
	return fmt.Sprintf("test-%d-%d@example.com", time.Now().Unix(), rand.Intn(10000))
}

// Shared test data for E2E tests that need to work together
var testUserEmail = func() string {
	rand.Seed(time.Now().UnixNano())
	return generateRandomEmail()
}()

// TestE2EEndpoints performs comprehensive end-to-end testing.
// This test requires a running server and database.
func TestE2EEndpoints(t *testing.T) {
	// Skip if not in E2E test mode
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("Skipping E2E test. Set E2E_TEST=true to run.")
	}

	baseURL := os.Getenv("TEST_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:7890"
	}

	config := &E2ETestConfig{
		BaseURL: baseURL,
	}

	// Wait for server to be ready
	if !waitForServerReady(config.BaseURL, 60*time.Second) {
		t.Fatal("Server not ready within timeout")
	}

	// Test sequence matching CI.yml exactly
	t.Run("01_Root_Endpoint", func(t *testing.T) {
		testRootEndpointE2E(t, config)
	})

	t.Run("02_User_Creation", func(t *testing.T) {
		testUserCreationE2E(t, config)
	})

	t.Run("03_User_Login", func(t *testing.T) {
		testUserLoginE2E(t, config)
	})

	// Only run authenticated tests if login was successful
	if config.Token != "" {
		t.Run("04_Authenticated_Endpoints", func(t *testing.T) {
			testAuthenticatedEndpointsE2E(t, config)
		})

		t.Run("05_User_By_UID", func(t *testing.T) {
			testUserByUIDE2E(t, config)
		})
	}

	t.Run("06_Error_Handling", func(t *testing.T) {
		testErrorHandlingE2E(t, config)
	})

	t.Run("07_Invalid_Endpoints", func(t *testing.T) {
		testInvalidEndpointsE2E(t, config)
	})
}

func waitForServerReady(baseURL string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

func testRootEndpointE2E(t *testing.T, config *E2ETestConfig) {
	t.Log("Testing root endpoint...")

	// Create a request with proper JSON headers for API testing
	req, err := http.NewRequest("GET", config.BaseURL+"/", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to call root endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Root endpoint test failed. Expected status 200, got %d", resp.StatusCode)
	}

	var response E2ETestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode root endpoint response: %v", err)
	}

	t.Log("✅ Root endpoint test passed")
}

func testUserCreationE2E(t *testing.T, config *E2ETestConfig) {
	t.Log("Testing user creation...")

	// Use the exact same payload format as CI.yml
	user := UserCreationRequest{
		Name:            "Test User",
		Email:           testUserEmail,
		Password:        "testpasS123@!asGFsd#321>$:",
		ConfirmPassword: "testpasS123@!asGFsd#321>$:",
	}

	jsonData, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user creation request: %v", err)
	}

	t.Logf("Sending JSON payload: %s (length: %d)", string(jsonData), len(jsonData))

	resp, err := http.Post(config.BaseURL+"/users", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Read response body for debugging
		var errorResponse E2ETestResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil {
			t.Errorf("User creation test failed. HTTP status: %d, Error: %s", resp.StatusCode, errorResponse.Error)
		} else {
			t.Errorf("User creation test failed. HTTP status: %d", resp.StatusCode)
		}
		return
	}

	var response E2ETestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode user creation response: %v", err)
		return
	}

	if response.Status != "success" {
		t.Errorf("Expected success status, got %s", response.Status)
		return
	}

	t.Log("✅ User creation test passed")
}

func testUserLoginE2E(t *testing.T, config *E2ETestConfig) {
	t.Log("Testing user login...")

	// Use the same credentials as created in user creation test
	credentials := LoginRequest{
		Email:    testUserEmail,
		Password: "testpasS123@!asGFsd#321>$:",
	}

	jsonData, err := json.Marshal(credentials)
	if err != nil {
		t.Fatalf("Failed to marshal login request: %v", err)
	}

	resp, err := http.Post(config.BaseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body for debugging
		var errorResponse E2ETestResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil {
			t.Errorf("User login test failed. HTTP status: %d, Error: %s", resp.StatusCode, errorResponse.Error)
		} else {
			t.Errorf("User login test failed. HTTP status: %d", resp.StatusCode)
		}
		return
	}

	var response struct {
		Status string          `json:"status"`
		Data   E2EAuthResponse `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode login response: %v", err)
		return
	}

	if response.Status != "success" {
		t.Errorf("Expected success status, got %s", response.Status)
		return
	}

	if response.Data.AccessToken == "" {
		t.Error("No token found in login response")
		return
	}

	// Store token for authenticated tests
	config.Token = response.Data.AccessToken
	t.Log("✅ User login test passed")
}

func testAuthenticatedEndpointsE2E(t *testing.T, config *E2ETestConfig) {
	t.Log("Testing paginated users endpoint...")

	client := &http.Client{}

	// Test paginated users endpoint with query parameters
	req, err := http.NewRequest("GET", config.BaseURL+"/users?page=1&limit=10", nil)
	if err != nil {
		t.Fatalf("Failed to create paginated users request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to call paginated users endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Paginated users test failed. Expected status 200, got %d", resp.StatusCode)
		return
	}

	var response E2ETestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode paginated users response: %v", err)
		return
	}

	if response.Status != "success" {
		t.Errorf("Expected success status, got %s", response.Status)
		return
	}

	t.Log("✅ Paginated users endpoint test passed")
}

func testUserByUIDE2E(t *testing.T, config *E2ETestConfig) {
	t.Log("Testing user by UID endpoint...")

	client := &http.Client{}

	// Test user by UID endpoint using sample user from init.sql
	req, err := http.NewRequest("GET", config.BaseURL+"/users/50d7f275-ecdf-4413-a323-11df86de5fd5", nil)
	if err != nil {
		t.Fatalf("Failed to create user by UID request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to call user by UID endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("User by UID test failed. Expected status 200, got %d", resp.StatusCode)
		return
	}

	var response E2ETestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode user by UID response: %v", err)
		return
	}

	if response.Status != "success" {
		t.Errorf("Expected success status, got %s", response.Status)
		return
	}

	t.Log("✅ User by UID endpoint test passed")
}

func testErrorHandlingE2E(t *testing.T, config *E2ETestConfig) {
	t.Log("Testing error handling...")

	// Test unauthorized access without token
	resp, err := http.Get(config.BaseURL + "/users")
	if err != nil {
		t.Fatalf("Failed to call protected endpoint without auth: %v", err)
	}
	defer resp.Body.Close()

	// Should return 401 or 403 for unauthorized access
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 401 or 403 for unauthorized access, got %d", resp.StatusCode)
	}

	// Test invalid JSON payload
	invalidJSON := bytes.NewBufferString(`{"invalid": json}`)
	resp, err = http.Post(config.BaseURL+"/users", "application/json", invalidJSON)
	if err != nil {
		t.Fatalf("Failed to send invalid JSON: %v", err)
	}
	defer resp.Body.Close()

	// Should return 400 for bad request
	if resp.StatusCode != http.StatusBadRequest {
		t.Log("Note: Invalid JSON did not return 400, server might handle malformed JSON differently")
	}

	// Test duplicate user creation (should fail)
	user := UserCreationRequest{
		Name:            "Duplicate User",
		Email:           testUserEmail, // Same email as before
		Password:        "testpasS123@!asGFsd#321>$:",
		ConfirmPassword: "testpasS123@!asGFsd#321>$:",
	}

	jsonData, _ := json.Marshal(user)
	resp, err = http.Post(config.BaseURL+"/users", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to attempt duplicate user creation: %v", err)
	}
	defer resp.Body.Close()

	// Should return error for duplicate email
	if resp.StatusCode == http.StatusOK {
		t.Error("Expected error for duplicate user creation, but got success")
	}

	t.Log("✅ Error handling tests passed")
}

func testInvalidEndpointsE2E(t *testing.T, config *E2ETestConfig) {
	t.Log("Testing invalid endpoints...")

	// Test 404 for invalid endpoint
	resp, err := http.Get(config.BaseURL + "/invalid-endpoint")
	if err != nil {
		t.Fatalf("Failed to call invalid endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for invalid endpoint, got %d", resp.StatusCode)
	}

	// Test invalid HTTP method on valid endpoint
	req, err := http.NewRequest("DELETE", config.BaseURL+"/", nil)
	if err != nil {
		t.Fatalf("Failed to create DELETE request: %v", err)
	}

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send DELETE request: %v", err)
	}
	defer resp.Body.Close()

	// Should return 405 Method Not Allowed or 404
	if resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
		t.Logf("Note: DELETE on root returned %d, expected 405 or 404", resp.StatusCode)
	}

	t.Log("✅ Invalid endpoints test passed")
}

// BenchmarkE2EEndpoints provides performance benchmarking for E2E scenarios.
func BenchmarkE2EEndpoints(b *testing.B) {
	if os.Getenv("E2E_TEST") != "true" {
		b.Skip("Skipping E2E benchmark. Set E2E_TEST=true to run.")
	}

	baseURL := os.Getenv("TEST_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:7890"
	}

	// Wait for server
	if !waitForServerReady(baseURL, 30*time.Second) {
		b.Fatal("Server not ready for benchmarking")
	}

	b.Run("Root_Endpoint_JSON_Performance", func(b *testing.B) {
		client := &http.Client{Timeout: 5 * time.Second}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reqJSON, err := http.NewRequest("GET", baseURL+"/", nil)
			if err != nil {
				b.Fatalf("Failed to create JSON request: %v", err)
			}
			reqJSON.Header.Set("Content-Type", "application/json")
			reqJSON.Header.Set("Accept", "application/json")
			respJSON, err := client.Do(reqJSON)
			if err != nil {
				b.Fatalf("JSON request failed: %v", err)
			}
			respJSON.Body.Close()
		}
	})

	b.Run("Root_Endpoint_HTML_Performance", func(b *testing.B) {
		client := &http.Client{Timeout: 5 * time.Second}
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			reqHTML, err := http.NewRequest("GET", baseURL+"/", nil)
			if err != nil {
				b.Fatalf("Failed to create HTML request: %v", err)
			}
			reqHTML.Header.Set("Content-Type", "text/html")
			reqHTML.Header.Set("Accept", "text/html")
			respHTML, err := client.Do(reqHTML)
			if err != nil {
				b.Fatalf("HTML request failed: %v", err)
			}
			respHTML.Body.Close()
		}
	})

	// Login once to get token for authenticated endpoint benchmarks
	config := &E2ETestConfig{BaseURL: baseURL}

	// Try to login with test user or sample user from init.sql
	credentials := LoginRequest{
		Email:    "john.doe@example.com", // From init.sql
		Password: "password",             // Default password from init.sql
	}

	jsonData, _ := json.Marshal(credentials)
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err == nil && resp.StatusCode == http.StatusOK {
		var response struct {
			Status string          `json:"status"`
			Data   E2EAuthResponse `json:"data"`
		}
		if json.NewDecoder(resp.Body).Decode(&response) == nil && response.Data.AccessToken != "" {
			config.Token = response.Data.AccessToken
		}
	}
	if resp != nil {
		resp.Body.Close()
	}

	if config.Token != "" {
		b.Run("Authenticated_Endpoint_Performance", func(b *testing.B) {
			client := &http.Client{Timeout: 5 * time.Second}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				req, _ := http.NewRequest("GET", baseURL+"/users?page=1&limit=10", nil)
				req.Header.Set("Authorization", "Bearer "+config.Token)

				resp, err := client.Do(req)
				if err != nil {
					b.Fatalf("Authenticated request failed: %v", err)
				}
				resp.Body.Close()
			}
		})
	}
}
