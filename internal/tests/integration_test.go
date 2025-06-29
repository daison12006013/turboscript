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
	"net/http"
	"os"
	"testing"
	"time"
)

// IntegrationTestConfig holds configuration for integration tests.
type IntegrationTestConfig struct {
	BaseURL string
	Token   string
}

// TestResponse represents a typical API response structure.
type TestResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data"`
	Error  string `json:"error,omitempty"`
}

// TestUser represents a user for testing.
type TestUser struct {
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AuthResponse represents login response.
type AuthResponse struct {
	UID                   string `json:"uid"`
	Name                  string `json:"name"`
	Email                 string `json:"email"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
	PasswordNeedsRehash   bool   `json:"password_needs_rehash"`
}

// TestIntegrationEndpoints tests all endpoints with proper flow.
// This test requires a running server and database.
func TestIntegrationEndpoints(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run.")
	}

	baseURL := os.Getenv("TEST_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:7890"
	}

	config := &IntegrationTestConfig{
		BaseURL: baseURL,
	}

	// Wait for server to be ready
	if !waitForServer(config.BaseURL, 30*time.Second) {
		t.Fatal("Server not ready within timeout")
	}

	// Test 1: Root endpoint
	t.Run("Root endpoint", func(t *testing.T) {
		testRootEndpoint(t, config)
	})

	// Test 2: User registration
	t.Run("User registration", func(t *testing.T) {
		testUserRegistration(t, config)
	})

	// Test 3: User login
	t.Run("User login", func(t *testing.T) {
		testUserLogin(t, config)
	})

	// Test 4: Authenticated endpoints (requires login)
	if config.Token != "" {
		t.Run("Authenticated endpoints", func(t *testing.T) {
			testAuthenticatedEndpoints(t, config)
		})
	}

	// Test 5: Error handling
	t.Run("Error handling", func(t *testing.T) {
		testErrorHandling(t, config)
	})
}

func waitForServer(baseURL string, timeout time.Duration) bool {
	client := &http.Client{Timeout: 1 * time.Second}
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
		time.Sleep(1 * time.Second)
	}
	return false
}

func testRootEndpoint(t *testing.T, config *IntegrationTestConfig) {
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
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var response TestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
}

func testUserRegistration(t *testing.T, config *IntegrationTestConfig) {
	// Use the accurate format from CI.yml with confirm_password
	user := map[string]string{
		"name":             "Integration Test User",
		"email":            fmt.Sprintf("integration-test-%d@example.com", time.Now().Unix()),
		"password":         "integrationPass123@",
		"confirm_password": "integrationPass123@",
	}

	jsonData, _ := json.Marshal(user)
	resp, err := http.Post(config.BaseURL+"/users", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Read response body for debugging
		var errorResponse TestResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil {
			t.Errorf("Expected status 201 for user registration, got %d. Error: %s", resp.StatusCode, errorResponse.Error)
		} else {
			t.Errorf("Expected status 201 for user registration, got %d", resp.StatusCode)
		}
		return
	}

	var response TestResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode registration response: %v", err)
		return
	}

	if response.Status != "success" {
		t.Errorf("Expected success status, got %s", response.Status)
	}
}

func testUserLogin(t *testing.T, config *IntegrationTestConfig) {
	// Try to login with sample user from init.sql
	credentials := map[string]string{
		"email":    "john.doe@example.com",
		"password": "password", // Default password from init.sql
	}

	jsonData, _ := json.Marshal(credentials)
	resp, err := http.Post(config.BaseURL+"/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for login, got %d", resp.StatusCode)
		return
	}

	var response struct {
		Status string       `json:"status"`
		Data   AuthResponse `json:"data"`
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
		t.Error("Expected token in login response")
		return
	}

	// Store token for authenticated tests
	config.Token = response.Data.AccessToken
}

func testAuthenticatedEndpoints(t *testing.T, config *IntegrationTestConfig) {
	client := &http.Client{}

	// Test paginated users endpoint
	req, err := http.NewRequest("GET", config.BaseURL+"/users?page=1&limit=10", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to call users endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for users endpoint, got %d", resp.StatusCode)
	}

	// Test user by UID endpoint (using sample user from init.sql)
	req, err = http.NewRequest("GET", config.BaseURL+"/users/50d7f275-ecdf-4413-a323-11df86de5fd5", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+config.Token)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to call user by UID endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for user by UID endpoint, got %d", resp.StatusCode)
	}
}

func testErrorHandling(t *testing.T, config *IntegrationTestConfig) {
	// Test 404 for invalid endpoint
	resp, err := http.Get(config.BaseURL + "/invalid-endpoint")
	if err != nil {
		t.Fatalf("Failed to call invalid endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for invalid endpoint, got %d", resp.StatusCode)
	}

	// Test unauthorized access
	resp, err = http.Get(config.BaseURL + "/users")
	if err != nil {
		t.Fatalf("Failed to call protected endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unauthorized access, got %d", resp.StatusCode)
	}

	// Test invalid JSON
	invalidJSON := bytes.NewBufferString(`{"invalid": json}`)
	resp, err = http.Post(config.BaseURL+"/users", "application/json", invalidJSON)
	if err != nil {
		t.Fatalf("Failed to send invalid JSON: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
	}
}

// BenchmarkEndpoints provides basic performance benchmarking.
func BenchmarkEndpoints(b *testing.B) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		b.Skip("Skipping benchmark. Set INTEGRATION_TEST=true to run.")
	}

	baseURL := os.Getenv("TEST_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:7890"
	}

	// Wait for server
	if !waitForServer(baseURL, 10*time.Second) {
		b.Fatal("Server not ready")
	}

	b.Run("Root endpoint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resp, err := http.Get(baseURL + "/")
			if err != nil {
				b.Fatalf("Request failed: %v", err)
			}
			resp.Body.Close()
		}
	})
}
