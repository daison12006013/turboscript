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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildDist tests the make build-dist command comprehensively.
func TestBuildDist(t *testing.T) {
	// Skip if not in build test mode
	if os.Getenv("BUILD_TEST") != "true" {
		t.Skip("Skipping build test. Set BUILD_TEST=true to run.")
	}

	// Get the root directory (go up two levels from internal/tests/)
	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get root directory: %v", err)
	}

	// Change to root directory for running make command
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()

	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}

	// Clean up any existing dist directory
	distDir := "dist"
	if err := os.RemoveAll(distDir); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to clean dist directory: %v", err)
	}

	// Run make build-dist
	t.Log("Running make build-dist...")
	cmd := exec.Command("make", "build-dist")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make build-dist failed: %v\nOutput: %s", err, output)
	}
	t.Logf("Build output: %s", output)

	// Test 1: Verify dist directory structure
	t.Run("DistDirectoryStructure", func(t *testing.T) {
		testDistDirectoryStructure(t, distDir)
	})

	// Test 2: Verify binary files
	t.Run("BinaryFiles", func(t *testing.T) {
		testBinaryFiles(t, distDir)
	})

	// Test 3: Verify TypeScript compilation
	t.Run("TypeScriptCompilation", func(t *testing.T) {
		testTypeScriptCompilation(t, distDir)
	})

	// Test 4: Verify configuration modifications
	t.Run("ConfigurationModifications", func(t *testing.T) {
		testConfigurationModifications(t, distDir)
	})

	// Test 5: Verify runner script
	t.Run("RunnerScript", func(t *testing.T) {
		testRunnerScript(t, distDir)
	})

	// Test 6: Test binary functionality
	t.Run("BinaryFunctionality", func(t *testing.T) {
		testBinaryFunctionality(t, distDir)
	})
}

func testDistDirectoryStructure(t *testing.T, distDir string) {
	expectedDirs := []string{
		"app",
		"app/routes",
		"app/routes/auth",
		"app/routes/users",
		"app/utils",
	}

	expectedFiles := []string{
		"turboscript",
		"turboscript-linux",
		"turboscript.yml",
		"runner.sh",
		"app/global.d.ts",
		"app/routes/index.js",
		"app/routes/auth/login.js",
		"app/routes/auth/logout.js",
		"app/routes/auth/refresh.js",
		"app/routes/users/create.js",
		"app/routes/users/filter-by-uid.js",
		"app/routes/users/paginated.js",
		"app/routes/users/change-password.js",
		"app/utils/cookies.js",
		"app/utils/jwt.js",
		"app/utils/meta.js",
		"app/utils/password.js",
	}

	// Check directories
	for _, dir := range expectedDirs {
		fullPath := filepath.Join(distDir, dir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected directory %s does not exist", fullPath)
		}
	}

	// Check files
	for _, file := range expectedFiles {
		fullPath := filepath.Join(distDir, file)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", fullPath)
		}
	}
}

func testBinaryFiles(t *testing.T, distDir string) {
	binaries := []string{"turboscript", "turboscript-linux"}

	for _, binary := range binaries {
		binaryPath := filepath.Join(distDir, binary)

		// Check if file exists
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			t.Errorf("Binary %s does not exist", binaryPath)
			continue
		}

		// Check if file is executable (for non-Windows systems)
		if info, err := os.Stat(binaryPath); err == nil {
			if info.Mode()&0111 == 0 {
				t.Errorf("Binary %s is not executable", binaryPath)
			}
		}

		// Check file size (should be > 1MB for Go binary)
		if info, err := os.Stat(binaryPath); err == nil {
			if info.Size() < 1024*1024 {
				t.Errorf("Binary %s seems too small (%d bytes)", binaryPath, info.Size())
			}
		}
	}
}

func testTypeScriptCompilation(t *testing.T, distDir string) {
	// Check that TypeScript declaration files are preserved
	globalDtsPath := filepath.Join(distDir, "app/global.d.ts")
	if _, err := os.Stat(globalDtsPath); os.IsNotExist(err) {
		t.Errorf("TypeScript declaration file %s was not preserved", globalDtsPath)
	}
}

func testConfigurationModifications(t *testing.T, distDir string) {
	configPath := filepath.Join(distDir, "turboscript.yml")

	// Read the configuration file
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file %s: %v", configPath, err)
	}

	configStr := string(content)

	// Check that debug mode is disabled
	if strings.Contains(configStr, "debug: true") {
		t.Error("Configuration still has debug: true (should be debug: false)")
	}

	// Check that monitoring is disabled
	if strings.Contains(configStr, "monitoring: true") {
		t.Error("Configuration still has monitoring: true (should be monitoring: false)")
	}

	// Verify production settings
	if !strings.Contains(configStr, "debug: false") {
		t.Error("Configuration doesn't have debug: false")
	}

	if !strings.Contains(configStr, "monitoring: false") {
		t.Error("Configuration doesn't have monitoring: false")
	}
}

func testRunnerScript(t *testing.T, distDir string) {
	runnerPath := filepath.Join(distDir, "runner.sh")

	// Check if runner script exists
	if _, err := os.Stat(runnerPath); os.IsNotExist(err) {
		t.Fatalf("Runner script %s does not exist", runnerPath)
	}

	// Check if script is executable
	if info, err := os.Stat(runnerPath); err == nil {
		if info.Mode()&0111 == 0 {
			t.Error("Runner script is not executable")
		}
	}

	// Read and verify script content
	content, err := os.ReadFile(runnerPath)
	if err != nil {
		t.Fatalf("Failed to read runner script: %v", err)
	}

	scriptStr := string(content)

	// Check for shebang
	if !strings.HasPrefix(scriptStr, "#!/bin/bash") {
		t.Error("Runner script missing proper shebang")
	}

	// Check for database configuration comment
	if !strings.Contains(scriptStr, "database connections") {
		t.Error("Runner script doesn't contain database configuration guidance")
	}

	// Check for turboscript executable
	if !strings.Contains(scriptStr, "./turboscript") {
		t.Error("Runner script doesn't contain turboscript executable")
	}
}

func testBinaryFunctionality(t *testing.T, distDir string) {
	binaryPath := filepath.Join(distDir, "turboscript")

	// Test help command
	t.Run("HelpCommand", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "help")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Errorf("Help command failed: %v\nOutput: %s", err, output)
			return
		}

		outputStr := string(output)
		expectedHelpTexts := []string{
			"TurboScript",
			"Usage:",
			"Start the server",
			"options:",
		}

		for _, expected := range expectedHelpTexts {
			if !strings.Contains(strings.ToLower(outputStr), strings.ToLower(expected)) {
				t.Errorf("Help output missing expected text: %s", expected)
			}
		}
	})

	// Test version-like information (if available)
	t.Run("BasicInformation", func(t *testing.T) {
		// Try to get some basic information without starting the server
		cmd := exec.Command(binaryPath, "metrics")
		output, err := cmd.CombinedOutput()

		// This might fail if no server is running, but it shouldn't crash
		if err != nil {
			outputStr := string(output)
			// Should contain some meaningful error message, not just crash
			if len(outputStr) == 0 {
				t.Error("Binary crashed without any output")
			}
		}
	})
}

// TestBuildDistCleanup tests that the build process cleans up properly.
func TestBuildDistCleanup(t *testing.T) {
	if os.Getenv("BUILD_TEST") != "true" {
		t.Skip("Skipping build cleanup test. Set BUILD_TEST=true to run.")
	}

	// Get the root directory (go up two levels from internal/tests/)
	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to get root directory: %v", err)
	}

	// Change to root directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()

	if err := os.Chdir(rootDir); err != nil {
		t.Fatalf("Failed to change to root directory: %v", err)
	}

	// Check if we can clean up the dist directory
	distDir := "dist"

	// Create the directory first (should exist from previous test)
	if _, err := os.Stat(distDir); os.IsNotExist(err) {
		t.Skip("Dist directory doesn't exist, skipping cleanup test")
	}

	// Test that we can remove the dist directory
	if err := os.RemoveAll(distDir); err != nil {
		t.Errorf("Failed to clean up dist directory: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(distDir); !os.IsNotExist(err) {
		t.Error("Dist directory still exists after cleanup")
	}
}

// BenchmarkBuildDist benchmarks the build process.
func BenchmarkBuildDist(b *testing.B) {
	if os.Getenv("BUILD_TEST") != "true" {
		b.Skip("Skipping build benchmark. Set BUILD_TEST=true to run.")
	}

	// Get the root directory (go up two levels from internal/tests/)
	rootDir, err := filepath.Abs("../..")
	if err != nil {
		b.Fatalf("Failed to get root directory: %v", err)
	}

	// Change to root directory
	originalDir, err := os.Getwd()
	if err != nil {
		b.Fatalf("Failed to get current directory: %v", err)
	}
	defer func() {
		os.Chdir(originalDir)
	}()

	if err := os.Chdir(rootDir); err != nil {
		b.Fatalf("Failed to change to root directory: %v", err)
	}

	for i := 0; i < b.N; i++ {
		// Clean up
		os.RemoveAll("dist")

		// Time the build process
		cmd := exec.Command("make", "build-dist")
		if err := cmd.Run(); err != nil {
			b.Fatalf("Build failed: %v", err)
		}
	}
}
