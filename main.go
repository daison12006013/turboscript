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

// Package main provides the TurboScript runtime executable.
//
// TurboScript is a hybrid web framework that combines TypeScript for business logic
// and Go for runtime execution. It uses JavaScript VM (goja) to execute TypeScript
// code at runtime, providing a unique development experience where TypeScript defines
// the API logic and Go handles the execution engine.
//
// The main package serves as the entry point for the TurboScript runtime, providing:
//   - Command-line interface for server operations
//   - Performance profiling and monitoring capabilities
//   - Database connection management with multi-connection support
//   - Server initialization and lifecycle management
//
// Usage:
//
//	turboscript                     Start the server
//	turboscript profile [options]   Collect performance profiles
//	turboscript metrics             Show performance metrics
//	turboscript help                Show help information
//
// Database Configuration:
//   - turboscript.yml: Configure database connections with environment variable support
//
// Configuration:
//   - turboscript.yml: Main configuration file defining routes, database connections, and logging
//
// For more information, visit: https://github.com/daison12006013/turboscript
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/email"
	"github.com/daison12006013/turboscript/internal/jobs"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/performance"
	"github.com/daison12006013/turboscript/internal/plugins"
	"github.com/daison12006013/turboscript/internal/server"

	_ "github.com/lib/pq"
)

const jsonFormat = "json"

func main() {
	// Global panic recovery to prevent crashes
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🚨 PANIC RECOVERED: %v", r)
			log.Printf("Application will continue running...")
			debug.PrintStack()
		}
	}()

	// Check for performance monitoring commands first (before flag parsing)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "profile":
			handleProfileCommand()
			return
		case "metrics":
			handleMetricsCommand()
			return
		case "help":
			printUsage()
			return
		}
	}

	// Default server startup
	startServer()
}

func printUsage() {
	fmt.Println("TurboScript - High-performance TypeScript+Go web framework")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  turboscript                     Start the server")
	fmt.Println("  turboscript profile [options]   Collect performance profiles")
	fmt.Println("  turboscript metrics             Show performance metrics")
	fmt.Println("  turboscript help                Show this help")
	fmt.Println()
	fmt.Println("Server options:")
	fmt.Println("  --config=FILE                   Configuration file path (default: turboscript.yml)")
	fmt.Println("                                  Auto-detects turboscript.dev.yml in Docker environments")
	fmt.Println()
	fmt.Println("Database Configuration:")
	fmt.Println("  Configure database connections in turboscript.yml:")
	fmt.Println("  database:")
	fmt.Println("    default: \"main\"")
	fmt.Println("    connections:")
	fmt.Println("      main:")
	fmt.Println("        driver: \"postgres\"")
	fmt.Println("        host: ${env:DB_HOST,\"localhost\"}")
	fmt.Println("        port: 5432")
	fmt.Println("        username: ${env:DB_USERNAME,\"user\"}")
	fmt.Println("        password: ${env:DB_PASSWORD,\"pass\"}")
	fmt.Println("        database: ${env:DB_NAME,\"mydb\"}")
	fmt.Println()
	fmt.Println("Profile options:")
	fmt.Println("  cpu [seconds]                   Collect CPU profile (default: 30s)")
	fmt.Println("  memory                          Collect memory profile")
	fmt.Println("  goroutine                       Collect goroutine profile")
	fmt.Println("  trace [seconds]                 Collect execution trace (default: 5s)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  turboscript profile cpu 60      # 60-second CPU profile")
	fmt.Println("  turboscript profile memory      # Memory heap profile")
	fmt.Println("  turboscript metrics             # Show current metrics")
}

func handleProfileCommand() {
	if len(os.Args) < 3 {
		fmt.Println("Profile command requires a type. Use 'turboscript help' for usage.")
		return
	}

	pm := performance.NewProfileManager("")

	switch os.Args[2] {
	case "cpu":
		duration := 30
		if len(os.Args) > 3 {
			if d, err := strconv.Atoi(os.Args[3]); err == nil {
				duration = d
			}
		}
		if err := pm.GetCPUProfile(duration); err != nil {
			fmt.Printf("CPU profile failed: %v\n", err)
		}
	case "memory":
		if err := pm.GetMemoryProfile(); err != nil {
			fmt.Printf("Memory profile failed: %v\n", err)
		}
	case "goroutine":
		if err := pm.GetGoroutineProfile(); err != nil {
			fmt.Printf("Goroutine profile failed: %v\n", err)
		}
	case "trace":
		duration := 5
		if len(os.Args) > 3 {
			if d, err := strconv.Atoi(os.Args[3]); err == nil {
				duration = d
			}
		}
		if err := pm.GetTrace(duration); err != nil {
			fmt.Printf("Execution trace failed: %v\n", err)
		}
	default:
		fmt.Printf("Unknown profile type: %s\n", os.Args[2])
		fmt.Println("Available types: cpu, memory, goroutine, trace")
	}
}

func handleMetricsCommand() {
	format := "detailed"
	if len(os.Args) > 2 && os.Args[2] == jsonFormat {
		format = jsonFormat
	}

	if format == jsonFormat {
		performance.PrintMetricsJSON()
	} else {
		performance.PrintDetailedReport()
	}
}

// determineConfigPath determines which configuration file to use based on priority:
// 1. Command line flag --config
// 2. Environment variable TURBOSCRIPT_CONFIG
// 3. turboscript.dev.yml if running in Docker (detected by DOCKER_ENV or other indicators)
// 4. turboscript.yml (default).
func determineConfigPath(flagConfig string) string {
	// Priority 1: Command line flag
	if flagConfig != "" {
		return flagConfig
	}

	// Priority 2: Environment variable
	if envConfig := os.Getenv("TURBOSCRIPT_CONFIG"); envConfig != "" {
		return envConfig
	}

	// Priority 3: Check if running in Docker environment
	if isDockerEnvironment() {
		devConfigPath := "turboscript.dev.yml"
		if _, err := os.Stat(devConfigPath); err == nil {
			return devConfigPath
		}
	}

	// Priority 4: Default configuration
	return "turboscript.yml"
}

// isDockerEnvironment detects if the application is running inside a Docker container.
func isDockerEnvironment() bool {
	// Check common Docker environment indicators
	dockerIndicators := []string{
		"DOCKER_ENV",       // Custom Docker environment variable
		"CONTAINER",        // Generic container indicator
		"IN_DOCKER",        // Common Docker flag
		"DOCKER_CONTAINER", // Another common Docker flag
	}

	for _, indicator := range dockerIndicators {
		if os.Getenv(indicator) != "" {
			return true
		}
	}

	// Check if we're running inside a container by looking at cgroup
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check for Docker-specific filesystem markers
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "container") {
			return true
		}
	}

	return false
}

func startServer() {
	// Server-specific panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🚨 SERVER PANIC RECOVERED: %v", r)
			log.Printf("Attempting to restart server...")
			// You could implement restart logic here if needed
		}
	}()

	// Parse command line flags
	configFile := flag.String("config", "", "Configuration file path (default: turboscript.yml, or turboscript.dev.yml in Docker)")
	flag.Parse()

	// Determine config file path
	configPath := determineConfigPath(*configFile)

	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Printf("Config error loading %s: %v", configPath, err)
		return
	}

	// Log which config file was loaded
	fmt.Printf("📋 Loaded configuration from: %s\n", configPath)

	// Initialize built-in plugins
	plugins.InitializeBuiltinPlugins()

	// Initialize plugins with configuration
	if err := plugins.InitializePluginsWithConfig(cfg.Plugins); err != nil {
		log.Printf("Failed to initialize plugins: %v", err)
		// Continue without plugins - they're optional
	} else {
		logger.Info("Plugins initialized successfully")
	}

	// Initialize logger with all logging settings
	logger.SetLoggingLevels(cfg.Debug, cfg.Info, cfg.Warning)

	// Set up file logging if specified
	if cfg.LogFile != "" {
		err = logger.SetLogFile(cfg.LogFile)
		if err != nil {
			logger.Error("Failed to set up log file %s: %v", cfg.LogFile, err)
			logger.Info("Continuing with console logging only")
		} else {
			logger.Info("Logging to file: %s", cfg.LogFile)
		}
	}

	// Log environment information for debugging
	logger.Debug("Configuration selection process:")
	logger.Debug("  - Command line flag: %s", *configFile)
	logger.Debug("  - TURBOSCRIPT_CONFIG env: %s", os.Getenv("TURBOSCRIPT_CONFIG"))
	logger.Debug("  - Docker environment detected: %t", isDockerEnvironment())
	logger.Debug("  - Selected config: %s", configPath)

	if cfg.Debug {
		logger.Info("Debug logging enabled")
	}
	if cfg.Info {
		logger.Debug("Info logging enabled")
	} else {
		logger.Debug("Info logging disabled")
	}
	if cfg.Warning {
		logger.Debug("Warning logging enabled")
	} else {
		logger.Debug("Warning logging disabled")
	}

	// Initialize database connections
	dbManager := config.NewDatabaseManager(&cfg.Database)
	if err := dbManager.InitializeConnections(); err != nil {
		log.Printf("Database connection initialization failed: %v", err)
		log.Printf("Please check your database configuration in turboscript.yml")
		log.Printf("For more information, see: https://github.com/daison12006013/turboscript/blob/main/README.md#database-configuration")
		return
	}
	defer func() {
		if closeErr := dbManager.Close(); closeErr != nil {
			log.Printf("Failed to close database connections: %v", closeErr)
		}
	}()

	// Get default database connection for jobs and other legacy components
	defaultDB, err := dbManager.GetDefaultConnection()
	if err != nil {
		log.Printf("Failed to get default database connection: %v", err)
		return
	}

	// Initialize email service
	emailService := email.NewServiceWithServerConfig(&cfg.Email, &cfg.Server)

	// Initialize job manager with database persistence and email service
	// No shared executor needed - each worker uses its own isolated executor
	jobManager := jobs.NewManagerWithEmail(&cfg.Jobs, &cfg.Cache, &cfg.Server, defaultDB, emailService)

	// Start job manager if enabled
	if cfg.Jobs.Enabled {
		if err := jobManager.Start(); err != nil {
			log.Printf("Failed to start job manager: %v", err)
			return
		}
		defer func() {
			logger.Info("Stopping job manager...")
			if stopErr := jobManager.Stop(); stopErr != nil {
				log.Printf("Failed to stop job manager: %v", stopErr)
			}
		}()
	}

	// Ensure log file is closed on shutdown
	defer logger.CloseLogFile()

	// Start server with job manager, email service, and database manager support
	srv := server.NewServerWithServices(cfg, dbManager, jobManager, emailService)
	srv.Start()
}
