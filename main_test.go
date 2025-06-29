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
	"testing"
)

func TestPrintUsage(t *testing.T) {
	// Capture stdout
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test doesn't crash
	printUsage()
}

func TestHandleProfileCommand(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "insufficient args",
			args: []string{"turboscript", "profile"},
		},
		{
			name: "unknown profile type",
			args: []string{"turboscript", "profile", "unknown"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			// Test that it doesn't panic
			handleProfileCommand()
		})
	}
}

func TestHandleMetricsCommand(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "default format",
			args: []string{"turboscript", "metrics"},
		},
		{
			name: "json format",
			args: []string{"turboscript", "metrics", "json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			// Test that it doesn't panic
			handleMetricsCommand()
		})
	}
}

func TestMainArgs(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		expected string // We can't easily test the actual behavior, just that it doesn't panic
	}{
		{
			name: "help command",
			args: []string{"turboscript", "help"},
		},
		{
			name: "profile command",
			args: []string{"turboscript", "profile"},
		},
		{
			name: "metrics command",
			args: []string{"turboscript", "metrics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			// We can't easily test main() without actually starting a server
			// but we can test the command parsing logic through the individual functions
			switch tt.args[1] {
			case "help":
				printUsage()
			case "profile":
				handleProfileCommand()
			case "metrics":
				handleMetricsCommand()
			}
		})
	}
}
