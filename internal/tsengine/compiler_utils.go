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

// Package tsengine provides TypeScript compilation utilities.
package tsengine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/evanw/esbuild/pkg/api"
)

// CompilerUtils provides utilities for TypeScript compilation.
type CompilerUtils struct {
	config *config.TypeScriptCompilerConfig
}

// NewCompilerUtils creates a new compiler utilities instance.
func NewCompilerUtils() *CompilerUtils {
	return &CompilerUtils{}
}

// NewCompilerUtilsWithConfig creates a new compiler utilities instance with configuration.
func NewCompilerUtilsWithConfig(cfg *config.TypeScriptCompilerConfig) *CompilerUtils {
	return &CompilerUtils{
		config: cfg,
	}
}

// ConvertTSToJS converts TypeScript to JavaScript with proper module resolution.
func (c *CompilerUtils) ConvertTSToJS(tsPath string) (string, error) {
	absPath, projectRoot, err := c.preparePaths(tsPath)
	if err != nil {
		return "", err
	}

	result := c.buildTypeScript(absPath, projectRoot)
	if err := c.handleBuildErrors(result); err != nil {
		return "", err
	}

	c.logBuildWarnings(result)

	if len(result.OutputFiles) == 0 {
		return "", fmt.Errorf("no output files generated from TypeScript compilation")
	}

	jsCode := string(result.OutputFiles[0].Contents)
	logger.Debug("Successfully compiled TypeScript to JavaScript (%d bytes)", len(jsCode))

	return jsCode, nil
}

// getExternalModules returns the external modules list from config or defaults.
func (c *CompilerUtils) getExternalModules() []string {
	if c.config != nil && len(c.config.ExternalModules) > 0 {
		return c.config.ExternalModules
	}
	// Default external modules
	return []string{
		"bcryptjs", "crypto", "node:crypto", "node:url", "node:assert", "node:util",
		"fs", "path", "os", "argon2",
	}
}

// getTarget returns the compilation target from config or default.
func (c *CompilerUtils) getTarget() api.Target {
	if c.config != nil && c.config.Target != "" {
		switch strings.ToUpper(c.config.Target) {
		case "ES2015":
			return api.ES2015
		case "ES2016":
			return api.ES2016
		case "ES2017":
			return api.ES2017
		case "ES2018":
			return api.ES2018
		case "ES2019":
			return api.ES2019
		case "ES2020":
			return api.ES2020
		case "ES2021":
			return api.ES2021
		case "ES2022":
			return api.ES2022
		default:
			logger.Warn("Unknown TypeScript target '%s', using ES2020", c.config.Target)
			return api.ES2020
		}
	}
	return api.ES2020 // Default
}

// getFormat returns the output format from config or default.
func (c *CompilerUtils) getFormat() api.Format {
	if c.config != nil && c.config.Format != "" {
		switch strings.ToUpper(c.config.Format) {
		case "COMMONJS", "CJS":
			return api.FormatCommonJS
		case "ESM", "ESMODULE", "MODULE":
			return api.FormatESModule
		case "IIFE":
			return api.FormatIIFE
		default:
			logger.Warn("Unknown TypeScript format '%s', using CommonJS", c.config.Format)
			return api.FormatCommonJS
		}
	}
	return api.FormatCommonJS // Default
}

// getSourcemap returns the sourcemap configuration from config.
func (c *CompilerUtils) getSourcemap() api.SourceMap {
	if c.config != nil && c.config.SourceMaps {
		return api.SourceMapInline
	}
	return api.SourceMapNone // Default: no sourcemaps
}

func (c *CompilerUtils) preparePaths(tsPath string) (string, string, error) {
	absPath, err := filepath.Abs(tsPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	projectRoot, err := c.findProjectRoot(filepath.Dir(absPath))
	if err != nil {
		logger.Debug("Could not find project root, using directory: %s", filepath.Dir(absPath))
		projectRoot = filepath.Dir(absPath)
	}

	return absPath, projectRoot, nil
}

func (c *CompilerUtils) buildTypeScript(absPath, projectRoot string) api.BuildResult {
	// Check if tsconfig.json exists
	tsconfigPath := filepath.Join(projectRoot, "tsconfig.json")
	_, err := os.Stat(tsconfigPath)
	tsconfigExists := err == nil

	// Determine external modules
	externalModules := c.getExternalModules()

	// Determine target
	target := c.getTarget()

	// Determine format
	format := c.getFormat()

	buildOptions := api.BuildOptions{
		EntryPoints: []string{absPath},
		Bundle:      true,
		Write:       false,
		Target:      target,
		Format:      format,
		Platform:    api.PlatformNode,
		Sourcemap:   c.getSourcemap(),
		Loader: map[string]api.Loader{
			".ts": api.LoaderTS,
		},
		Outdir:        "/tmp", // We don't actually write files
		AbsWorkingDir: projectRoot,
		NodePaths: []string{
			filepath.Join(projectRoot, "node_modules"),
		},
		ResolveExtensions: []string{".ts", ".js", ".json"},
		MainFields:        []string{"main", "module"},
		Conditions:        []string{"node"},
		External:          externalModules,
	}

	// Only set Tsconfig if the file exists
	if tsconfigExists {
		buildOptions.Tsconfig = tsconfigPath
	}

	return api.Build(buildOptions)
}

func (c *CompilerUtils) handleBuildErrors(result api.BuildResult) error {
	if len(result.Errors) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("TypeScript compilation errors:\n")
		for _, err := range result.Errors {
			errorMsg.WriteString(fmt.Sprintf("  - %s\n", err.Text))
		}
		return fmt.Errorf("%s", errorMsg.String())
	}
	return nil
}

func (c *CompilerUtils) logBuildWarnings(result api.BuildResult) {
	for _, warning := range result.Warnings {
		logger.Warn("TypeScript compilation warning: %s", warning.Text)
	}
}

func (c *CompilerUtils) findProjectRoot(startDir string) (string, error) {
	dir := startDir
	for {
		// Check for package.json or tsconfig.json
		packageJSON := filepath.Join(dir, "package.json")
		tsConfigJSON := filepath.Join(dir, "tsconfig.json")

		if _, err := os.Stat(packageJSON); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(tsConfigJSON); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("project root not found (no package.json or tsconfig.json found)")
}
