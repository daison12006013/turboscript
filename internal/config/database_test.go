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

package config

import (
	"os"
	"testing"
)

func TestDatabaseConfig_GetDefaultConnection(t *testing.T) {
	// Test case 1: Valid default connection
	dbConfig := DatabaseConfig{
		Default: "main",
		Connections: map[string]DatabaseConnection{
			"main": {
				Driver:   "postgres",
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "pass",
				Database: "testdb",
			},
		},
	}

	conn, err := dbConfig.GetDefaultConnection()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if conn.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", conn.Host)
	}

	// Test case 2: No default specified
	dbConfig.Default = ""
	_, err = dbConfig.GetDefaultConnection()
	if err == nil {
		t.Error("Expected error when no default connection specified")
	}

	// Test case 3: Default connection not found
	dbConfig.Default = "nonexistent"
	_, err = dbConfig.GetDefaultConnection()
	if err == nil {
		t.Error("Expected error when default connection not found")
	}
}

func TestDatabaseConnection_BuildConnectionString(t *testing.T) {
	conn := DatabaseConnection{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Username: "testuser",
		Password: "testpass",
		Database: "testdb",
		SSLMode:  "disable",
		Timezone: "UTC",
	}

	connStr, err := conn.BuildConnectionString()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expected := "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable&timezone=UTC"
	if connStr != expected {
		t.Errorf("Expected connection string '%s', got '%s'", expected, connStr)
	}

	// Test unsupported driver
	conn.Driver = "mysql"
	_, err = conn.BuildConnectionString()
	if err == nil {
		t.Error("Expected error for unsupported driver")
	}
}

func TestResolveEnvVariables(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_VAR", "test_value")
	os.Setenv("TEST_HOST", "test_host")
	defer func() {
		os.Unsetenv("TEST_VAR")
		os.Unsetenv("TEST_HOST")
	}()

	// Test case 1: Environment variable with no default
	result := resolveEnvVariables("${env:TEST_VAR}")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}

	// Test case 2: Environment variable with default value (env var exists)
	result = resolveEnvVariables("${env:TEST_VAR,\"default_value\"}")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", result)
	}

	// Test case 3: Environment variable with default value (env var doesn't exist)
	result = resolveEnvVariables("${env:NONEXISTENT_VAR,\"default_value\"}")
	if result != "default_value" {
		t.Errorf("Expected 'default_value', got '%s'", result)
	}

	// Test case 4: Multiple environment variables in same string
	result = resolveEnvVariables("postgres://${env:TEST_VAR}:pass@${env:TEST_HOST}:5432/db")
	expected := "postgres://test_value:pass@test_host:5432/db"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}

	// Test case 5: Regular string without env variables
	result = resolveEnvVariables("regular_string")
	if result != "regular_string" {
		t.Errorf("Expected 'regular_string', got '%s'", result)
	}
}

func TestResolveEnvVariables_MissingRequired(t *testing.T) {
	// Test case: Environment variable not found and no default provided
	// This should return an empty string (graceful behavior)
	result := resolveEnvVariables("${env:MISSING_REQUIRED_VAR}")
	if result != "" {
		t.Errorf("Expected empty string when environment variable is missing, got '%s'", result)
	}
}

func TestDatabaseConnection_BuildConnectionString_WithEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("DB_HOST", "prod_host")
	os.Setenv("DB_USER", "prod_user")
	os.Setenv("DB_PASS", "prod_pass")
	// Unset DB_NAME to ensure default value is used
	os.Unsetenv("DB_NAME")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASS")
		os.Unsetenv("DB_NAME")
	}()

	conn := DatabaseConnection{
		Driver:   "postgres",
		Host:     "${env:DB_HOST,\"localhost\"}",
		Port:     5432,
		Username: "${env:DB_USER,\"defaultuser\"}",
		Password: "${env:DB_PASS,\"defaultpass\"}",
		Database: "${env:DB_NAME,\"defaultdb\"}",
		SSLMode:  "disable",
		Timezone: "UTC",
	}

	connStr, err := conn.BuildConnectionString()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expected := "postgres://prod_user:prod_pass@prod_host:5432/defaultdb?sslmode=disable&timezone=UTC"
	if connStr != expected {
		t.Errorf("Expected connection string '%s', got '%s'", expected, connStr)
	}
}
