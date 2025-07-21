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

// Package config provides configuration loading and management for TurboScript.
//
// This package handles parsing and validation of the turboscript.yml configuration file,
// which defines routing, database access, logging levels, and other runtime settings.
//
// Configuration Structure:
//   - AllowedTables: List of database tables that can be accessed (security feature)
//   - Endpoints: API route definitions mapping HTTP methods/paths to TypeScript files
//   - Debug/Info/Warning: Logging level controls
//   - Monitoring: Performance monitoring toggle
//   - Port: HTTP server port
package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	trueValue   = "true"
	oneValue    = "1"
	defaultPort = 7890
)

// FolderConfig represents folder-specific configuration for file serving endpoints.
type FolderConfig struct {
	Type   string `yaml:"type"`   // Response type for folder endpoints (e.g., "markdown-html")
	Index  string `yaml:"index"`  // Default index file to serve when no specific file is requested
	Layout string `yaml:"layout"` // Layout file to wrap content in (relative to folder path)
}

// WebSocketChannelConfig represents a WebSocket channel configuration.
type WebSocketChannelConfig struct {
	Room           string `yaml:"room"`            // Room pattern (regex capable, e.g., "room-(.*)")
	Type           string `yaml:"type"`            // Channel type (public, private, presence)
	Handle         string `yaml:"handle"`          // Path to TypeScript handler file
	MaxConnections int    `yaml:"max_connections"` // Maximum connections (0 = unlimited)
}

// WebSocketConfig represents WebSocket-specific configuration.
type WebSocketConfig struct {
	Channels          []WebSocketChannelConfig `yaml:"channels"`           // Channel configurations
	EnablePresence    bool                     `yaml:"enable_presence"`    // Enable presence tracking
	EnableKafka       bool                     `yaml:"enable_kafka"`       // Enable Kafka integration for scaling
	EnableRedis       bool                     `yaml:"enable_redis"`       // Enable Redis for sticky sessions
	KafkaBrokers      []string                 `yaml:"kafka_brokers"`      // Kafka broker addresses
	KafkaTopic        string                   `yaml:"kafka_topic"`        // Kafka topic for WebSocket messages
	RedisChannel      string                   `yaml:"redis_channel"`      // Redis channel for pub/sub
	ReadBufferSize    int                      `yaml:"read_buffer_size"`   // WebSocket read buffer size (default: 1024)
	WriteBufferSize   int                      `yaml:"write_buffer_size"`  // WebSocket write buffer size (default: 1024)
	PingInterval      int                      `yaml:"ping_interval"`      // Ping interval in seconds (default: 30)
	PongTimeout       int                      `yaml:"pong_timeout"`       // Pong timeout in seconds (default: 60)
	MaxMessageSize    int64                    `yaml:"max_message_size"`   // Maximum message size in bytes (default: 512KB)
	CompressionLevel  int                      `yaml:"compression_level"`  // Compression level (0=no compression, 1-9=zlib levels)
	EnableCompression bool                     `yaml:"enable_compression"` // Enable per-message compression
	AllowedOrigins    []string                 `yaml:"allowed_origins"`    // CORS allowed origins for WebSocket connections
}

// GetReadBufferSize returns the read buffer size with a default value.
func (ws *WebSocketConfig) GetReadBufferSize() int {
	if ws.ReadBufferSize <= 0 {
		return 1024
	}
	return ws.ReadBufferSize
}

// GetWriteBufferSize returns the write buffer size with a default value.
func (ws *WebSocketConfig) GetWriteBufferSize() int {
	if ws.WriteBufferSize <= 0 {
		return 1024
	}
	return ws.WriteBufferSize
}

// GetPingInterval returns the ping interval in seconds with a default value.
func (ws *WebSocketConfig) GetPingInterval() int {
	if ws.PingInterval <= 0 {
		return 30
	}
	return ws.PingInterval
}

// GetPongTimeout returns the pong timeout in seconds with a default value.
func (ws *WebSocketConfig) GetPongTimeout() int {
	if ws.PongTimeout <= 0 {
		return 60
	}
	return ws.PongTimeout
}

// GetMaxMessageSize returns the maximum message size in bytes with a default value.
func (ws *WebSocketConfig) GetMaxMessageSize() int64 {
	if ws.MaxMessageSize <= 0 {
		return 512 * 1024 // 512KB
	}
	return ws.MaxMessageSize
}

// SSEConfig represents Server-Sent Events configuration.
type SSEConfig struct {
	EnableHTTP2       bool     `yaml:"enable_http2"`       // Force HTTP/2 for SSE (recommended)
	KeepAliveInterval int      `yaml:"keepalive_interval"` // Keep-alive interval in seconds (default: 30)
	Retry             int      `yaml:"retry"`              // Retry interval for client reconnection (default: 3000ms)
	MaxConnections    int      `yaml:"max_connections"`    // Maximum concurrent SSE connections (0 = unlimited)
	AllowedOrigins    []string `yaml:"allowed_origins"`    // CORS allowed origins for SSE
	BufferSize        int      `yaml:"buffer_size"`        // Buffer size for SSE messages (default: 1024)
	EnableCompression bool     `yaml:"enable_compression"` // Enable gzip compression for SSE
}

// GetKeepAliveInterval returns the keep-alive interval in seconds with a default value.
func (sse *SSEConfig) GetKeepAliveInterval() int {
	if sse.KeepAliveInterval <= 0 {
		return 30
	}
	return sse.KeepAliveInterval
}

// GetRetry returns the retry interval in milliseconds with a default value.
func (sse *SSEConfig) GetRetry() int {
	if sse.Retry <= 0 {
		return 3000
	}
	return sse.Retry
}

// GetBufferSize returns the buffer size with a default value.
func (sse *SSEConfig) GetBufferSize() int {
	if sse.BufferSize <= 0 {
		return 1024
	}
	return sse.BufferSize
}

// EndpointConfig represents a single API endpoint configuration.
//
// Each endpoint maps an HTTP method and path to a TypeScript file that handles the request.
// Optional table and operation fields can be used for additional metadata.
// Supports WebSocket and SSE endpoints for real-time communication.
type EndpointConfig struct {
	Route        string           `yaml:"route"`     // HTTP path pattern (e.g., "/users/{id}")
	Method       string           `yaml:"method"`    // HTTP method (GET, POST, PUT, DELETE, WebSocket, SSE)
	Path         string           `yaml:"path"`      // Path to TypeScript handler file or folder
	Type         string           `yaml:"type"`      // Endpoint type: "api" (default), "websocket", "sse", "markdown-html", "hybrid", etc.
	Options      map[string]any   `yaml:"options"`   // Type-specific options (index, layout, etc.)
	WebSocket    *WebSocketConfig `yaml:"websocket"` // WebSocket-specific configuration
	SSE          *SSEConfig       `yaml:"sse"`       // Server-Sent Events configuration
	Table        string           `yaml:"table"`     // Optional: primary database table for this endpoint
	Operation    string           `yaml:"operation"` // Optional: operation type (create, read, update, delete)
	Timeout      int              `yaml:"timeout"`   // Optional: endpoint-specific timeout in seconds (overrides global timeout)
	FolderConfig *FolderConfig    `yaml:"-"`         // Optional: folder-specific config for file serving endpoints
	Index        string           `yaml:"index"`     // Optional: default index file to serve when no specific file is requested
	Layout       string           `yaml:"layout"`    // Optional: layout file to wrap content in (relative to folder path)
	Markdown     bool             `yaml:"markdown"`  // Optional: enable Markdown rendering
}

// EmailConfig represents email configuration for different drivers.
type EmailConfig struct {
	DefaultDriver string         `yaml:"default_driver"` // Default email driver to use
	FromAddress   string         `yaml:"from_address"`   // Default from email address
	FromName      string         `yaml:"from_name"`      // Default from name
	SMTP          SMTPConfig     `yaml:"smtp"`           // SMTP configuration
	Mailgun       MailgunConfig  `yaml:"mailgun"`        // Mailgun configuration
	SES           SESConfig      `yaml:"ses"`            // AWS SES configuration
	SendGrid      SendGridConfig `yaml:"sendgrid"`       // SendGrid configuration
	Postmark      PostmarkConfig `yaml:"postmark"`       // Postmark configuration
}

// SMTPConfig represents SMTP email driver configuration.
type SMTPConfig struct {
	Host       string `yaml:"host"`       // SMTP server host
	Port       int    `yaml:"port"`       // SMTP server port
	Username   string `yaml:"username"`   // SMTP username
	Password   string `yaml:"password"`   // SMTP password
	Encryption string `yaml:"encryption"` // Encryption type (none, tls, ssl)
}

// MailgunConfig represents Mailgun email driver configuration.
type MailgunConfig struct {
	Domain string `yaml:"domain"`  // Mailgun domain
	APIKey string `yaml:"api_key"` // Mailgun API key
	Region string `yaml:"region"`  // Mailgun region (us, eu)
}

// SESConfig represents AWS SES email driver configuration.
type SESConfig struct {
	Region          string `yaml:"region"`            // AWS region
	AccessKeyID     string `yaml:"access_key_id"`     // AWS access key ID
	SecretAccessKey string `yaml:"secret_access_key"` // AWS secret access key
}

// SendGridConfig represents SendGrid email driver configuration.
type SendGridConfig struct {
	APIKey string `yaml:"api_key"` // SendGrid API key
}

// PostmarkConfig represents Postmark email driver configuration.
type PostmarkConfig struct {
	ServerToken string `yaml:"server_token"` // Postmark server token
}

// DatabaseConnection represents a single database connection configuration.
type DatabaseConnection struct {
	Driver             string `yaml:"driver"`               // Database driver (currently only "postgres")
	Host               string `yaml:"host"`                 // Database host
	Port               int    `yaml:"port"`                 // Database port
	Username           string `yaml:"username"`             // Database username
	Password           string `yaml:"password"`             // Database password
	Database           string `yaml:"database"`             // Database name
	MaxOpenConnections int    `yaml:"max_open_connections"` // Maximum number of open connections
	MaxIdleConnections int    `yaml:"max_idle_connections"` // Maximum number of idle connections
	ConnectionTimeout  int    `yaml:"connection_timeout"`   // Connection timeout in seconds
	MaxLifetime        int    `yaml:"max_lifetime"`         // Maximum connection lifetime in seconds
	MaxIdleTime        int    `yaml:"max_idle_time"`        // Maximum idle time in seconds
	SSLMode            string `yaml:"ssl_mode"`             // SSL mode (disable, require, verify-ca, verify-full)
	Timezone           string `yaml:"timezone"`             // Database timezone
}

// DatabaseConfig represents the complete database configuration with connections.
type DatabaseConfig struct {
	Default     string                        `yaml:"default"`     // Default connection name
	Connections map[string]DatabaseConnection `yaml:"connections"` // Named database connections
}

// GetDefaultConnection returns the default database connection configuration.
func (dc *DatabaseConfig) GetDefaultConnection() (*DatabaseConnection, error) {
	if dc.Default == "" {
		return nil, fmt.Errorf("no default database connection specified")
	}

	conn, exists := dc.Connections[dc.Default]
	if !exists {
		return nil, fmt.Errorf("default database connection '%s' not found in connections", dc.Default)
	}

	return &conn, nil
}

// getConnection returns a specific database connection configuration by name.
func (dc *DatabaseConfig) getConnection(name string) (*DatabaseConnection, error) {
	conn, exists := dc.Connections[name]
	if !exists {
		return nil, fmt.Errorf("database connection '%s' not found", name)
	}

	return &conn, nil
}

// BuildConnectionString builds a PostgreSQL connection string from the connection configuration.
func (conn *DatabaseConnection) BuildConnectionString() (string, error) {
	if conn.Driver != "postgres" {
		return "", fmt.Errorf("unsupported database driver: %s (only 'postgres' is currently supported)", conn.Driver)
	}

	// Create a copy of the connection and resolve all environment variables using the generic resolver
	resolvedConn := *conn // Create a copy
	if err := resolveStructEnvVariables(&resolvedConn); err != nil {
		return "", fmt.Errorf("failed to resolve environment variables: %w", err)
	}

	// Set defaults for optional fields
	if resolvedConn.SSLMode == "" {
		resolvedConn.SSLMode = "disable"
	}
	if resolvedConn.Timezone == "" {
		resolvedConn.Timezone = "UTC"
	}

	// Build PostgreSQL connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s&timezone=%s",
		resolvedConn.Username, resolvedConn.Password, resolvedConn.Host,
		resolvedConn.Port, resolvedConn.Database, resolvedConn.SSLMode, resolvedConn.Timezone)

	return connStr, nil
}

// resolveEnvVariables resolves environment variable references in the format ${env:VAR_NAME} or ${env:VAR_NAME, default_value}.
func resolveEnvVariables(value string) string {
	// Pattern to match ${env:VAR_NAME} or ${env:VAR_NAME, default_value} (with or without quotes)
	envPattern := regexp.MustCompile(`\$\{env:([^,}]+)(?:,\s*(?:"([^"]*)"|([^}]*)))?\}`)

	return envPattern.ReplaceAllStringFunc(value, func(match string) string {
		submatches := envPattern.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		envVar := strings.TrimSpace(submatches[1])
		defaultValue := ""

		// Check for quoted default (group 2) or unquoted default (group 3)
		if len(submatches) > 2 && submatches[2] != "" {
			defaultValue = submatches[2] // quoted default
		} else if len(submatches) > 3 && submatches[3] != "" {
			defaultValue = strings.TrimSpace(submatches[3]) // unquoted default
		}

		envValue := os.Getenv(envVar)
		if envValue != "" {
			return envValue
		}

		return defaultValue
	})
}

// JobsConfig represents background jobs configuration.
type JobsConfig struct {
	Enabled            bool   `yaml:"enabled"`        // Enable background job processing
	MaxWorkers         int    `yaml:"max_workers"`    // Maximum number of worker goroutines
	RetryAttempts      int    `yaml:"retry_attempts"` // Number of retry attempts for failed jobs
	RetryDelay         int    `yaml:"retry_delay"`    // Delay between retry attempts in seconds
	Timeout            int    `yaml:"timeout"`        // Job timeout in seconds
	QueueSize          int    `yaml:"queue_size"`     // Maximum queue size
	PathBackgroundJobs string `yaml:"path"`           // Path to background jobs directory
}

// SchedulerTaskConfig represents a single scheduled task configuration.
type SchedulerTaskConfig struct {
	Name        string            `yaml:"name"`        // Task name (must be unique)
	Cron        string            `yaml:"cron"`        // Cron expression (e.g., "0 2 * * *" for daily at 2 AM)
	Handler     string            `yaml:"handler"`     // Path to TypeScript handler file
	Enabled     bool              `yaml:"enabled"`     // Whether the task is enabled (default: true)
	Timezone    string            `yaml:"timezone"`    // Timezone for cron execution (default: UTC)
	Timeout     int               `yaml:"timeout"`     // Task timeout in seconds (overrides global)
	Description string            `yaml:"description"` // Optional description for documentation
	Payload     map[string]any    `yaml:"payload"`     // Optional static payload to pass to the handler
	Environment map[string]string `yaml:"environment"` // Optional environment variables for this task
}

// SchedulerConfig represents the scheduler configuration.
type SchedulerConfig struct {
	Enabled  bool                  `yaml:"enabled"`   // Enable scheduler system (default: true)
	Timezone string                `yaml:"timezone"`  // Default timezone for all tasks (default: UTC)
	Path     string                `yaml:"path"`      // Path to scheduler handlers directory (default: ./app/schedulers)
	LogLevel string                `yaml:"log_level"` // Log level: debug, info, warn, error (default: info)
	Tasks    []SchedulerTaskConfig `yaml:"tasks"`     // List of scheduled tasks
}

// CacheDriverConfig represents a single cache driver configuration.
type CacheDriverConfig struct {
	Driver               string `yaml:"driver"`                 // Driver type: memory, redis, memcached, file
	Host                 string `yaml:"host"`                   // Host address for network drivers
	Port                 int    `yaml:"port"`                   // Port for network drivers
	Password             string `yaml:"password"`               // Password for authentication
	DB                   int    `yaml:"db"`                     // Database number (Redis only)
	MaxSize              int    `yaml:"max_size"`               // Maximum cache size in MB
	Expiration           int    `yaml:"expiration"`             // Default expiration time in seconds
	MaxIdleConnections   int    `yaml:"max_idle_connections"`   // Maximum idle connections
	MaxActiveConnections int    `yaml:"max_active_connections"` // Maximum active connections
	IdleTimeout          int    `yaml:"idle_timeout"`           // Idle timeout in seconds
	ReadTimeout          int    `yaml:"read_timeout"`           // Read timeout in seconds
	WriteTimeout         int    `yaml:"write_timeout"`          // Write timeout in seconds
	Root                 string `yaml:"root"`                   // Root directory for file driver
}

// CacheConfig represents the cache configuration.
type CacheConfig struct {
	Default string                       `yaml:"default"` // Default cache driver name
	Drivers map[string]CacheDriverConfig `yaml:"drivers"` // Named cache driver configurations
}

// CompressionConfig represents the response compression configuration.
//
// Gzip compression can significantly reduce bandwidth usage and improve response times
// for text-based content (JSON, HTML, CSS, JS). The trade-off is a small CPU overhead
// for compression processing.
type CompressionConfig struct {
	Enabled bool `yaml:"enabled"`  // Enable/disable gzip compression (default: true)
	MinSize int  `yaml:"min_size"` // Minimum response size in bytes to compress (default: 1024)
	Level   int  `yaml:"level"`    // Compression level 1-9 (1=fastest, 9=best, 6=balanced, default: 6)
}

// ServerConfig represents the HTTP server configuration.
type ServerConfig struct {
	PoolSize            int    `yaml:"pool_size"`             // Executor pool size for concurrent request processing (default: 10)
	ProfilerPort        string `yaml:"profiler_port"`         // Port for pprof profiler server (default: "6060")
	PerformanceInterval int    `yaml:"performance_interval"`  // Performance metrics collection interval in seconds (default: 30)
	HTTPTimeout         int    `yaml:"http_timeout"`          // HTTP client timeout in seconds for outbound requests (default: 30)
	InitialCleanupDelay int    `yaml:"initial_cleanup_delay"` // Initial cleanup delay in seconds for job manager (default: 30)
}

// PluginConfig represents the configuration for a single plugin.
type PluginConfig struct {
	Name    string         `yaml:"name"`    // Plugin name
	Enabled bool           `yaml:"enabled"` // Whether the plugin is enabled
	Options map[string]any `yaml:"options"` // Plugin-specific options
}

// TypeScriptCompilerConfig represents TypeScript compilation configuration.
type TypeScriptCompilerConfig struct {
	ExternalModules []string `yaml:"external_modules"` // Modules to exclude from bundling (keep as imports)
	Target          string   `yaml:"target"`           // TypeScript compilation target (ES2020, ES2022, etc.)
	Format          string   `yaml:"format"`           // Output format (CommonJS, ESM, etc.)
	MinifyJS        bool     `yaml:"minify_js"`        // Enable JavaScript minification
	SourceMaps      bool     `yaml:"source_maps"`      // Generate source maps for debugging
}

// Config represents the complete TurboScript configuration.
//
// This structure mirrors the turboscript.yml file format and provides
// all necessary settings for runtime operation.
type Config struct {
	Endpoints        []EndpointConfig         `yaml:"endpoints"`         // API route definitions
	Env              map[string]string        `yaml:"env"`               // Environment variables to inject into events
	Database         DatabaseConfig           `yaml:"database"`          // Database configuration with connections
	Cache            CacheConfig              `yaml:"cache"`             // Cache configuration
	Email            EmailConfig              `yaml:"email"`             // Email configuration
	Jobs             JobsConfig               `yaml:"jobs"`              // Background jobs configuration
	Scheduler        SchedulerConfig          `yaml:"scheduler"`         // Scheduler configuration
	Plugins          []PluginConfig           `yaml:"plugins"`           // Plugin configurations
	Server           ServerConfig             `yaml:"server"`            // Server configuration
	Compression      CompressionConfig        `yaml:"compression"`       // Response compression configuration
	TypeScript       TypeScriptCompilerConfig `yaml:"typescript"`        // TypeScript compiler configuration
	Debug            bool                     `yaml:"debug"`             // Enable debug logging
	Info             bool                     `yaml:"info"`              // Enable info logging
	Warning          bool                     `yaml:"warning"`           // Enable warning logging
	Monitoring       bool                     `yaml:"monitoring"`        // Enable performance monitoring
	Port             int                      `yaml:"port"`              // HTTP server port
	PreserveResponse bool                     `yaml:"preserve_response"` // Preserve JSON response field order (slower performance)
	Timeout          int                      `yaml:"timeout"`           // Global timeout in seconds for async operations (default: 30)
	PreferJS         bool                     `yaml:"prefer_js"`         // Prefer .js files over .js files (auto-detected if not set)
	PreferTS         bool                     `yaml:"prefer_ts"`         // Prefer .ts files over .js files (overrides prefer_js)
	NotFoundPage     string                   `yaml:"not_found_page"`    // Path to custom 404 page handler (e.g., "./app/routes/404")
	LogFile          string                   `yaml:"log_file"`          // Path to log file (optional, defaults to stdout)
}

// LoadConfig loads and parses the TurboScript configuration file.
//
// The function reads the YAML configuration file from the specified path,
// parses it into a Config struct, and applies environment variable overrides
// for certain settings.
//
// Environment Variables:
//   - PORT: Override the configured port number
//
// Returns an error if the file cannot be read or parsed.
func LoadConfig(path string) (*Config, error) {
	data, err := loadConfigFile(path)
	if err != nil {
		return nil, err
	}

	cfg, err := parseConfigData(data)
	if err != nil {
		return nil, err
	}

	applyEnvironmentOverrides(&cfg)
	applyDefaultSettings(&cfg, data)

	return &cfg, nil
}

// loadConfigFile reads and resolves environment variables in the config file.
func loadConfigFile(path string) ([]byte, error) {
	// #nosec G304: Path is controlled by application, not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Resolve environment variables in the raw YAML content before unmarshaling
	// This ensures that integer fields with env var placeholders are properly resolved
	resolvedYAML := resolveEnvVariables(string(data))
	return []byte(resolvedYAML), nil
}

// parseConfigData parses the YAML data into a Config struct and resolves all environment variables.
func parseConfigData(data []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	// Resolve environment variables for the entire configuration structure
	if err := resolveStructEnvVariables(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to resolve environment variables in configuration: %w", err)
	}

	return cfg, nil
}

// applyEnvironmentOverrides applies environment variable overrides for logging and monitoring settings.
func applyEnvironmentOverrides(cfg *Config) {
	if debugEnv := os.Getenv("TURBOSCRIPT_DEBUG"); debugEnv != "" {
		cfg.Debug = strings.ToLower(debugEnv) == trueValue || debugEnv == oneValue
	}

	if infoEnv := os.Getenv("TURBOSCRIPT_INFO"); infoEnv != "" {
		cfg.Info = strings.ToLower(infoEnv) == trueValue || infoEnv == oneValue
	}

	if warningEnv := os.Getenv("TURBOSCRIPT_WARNING"); warningEnv != "" {
		cfg.Warning = strings.ToLower(warningEnv) == trueValue || warningEnv == oneValue
	}

	if monitoringEnv := os.Getenv("TURBOSCRIPT_MONITORING"); monitoringEnv != "" {
		cfg.Monitoring = strings.ToLower(monitoringEnv) == trueValue || monitoringEnv == oneValue
	}

	// Override port from environment variable
	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if port, err := strconv.Atoi(portEnv); err == nil {
			cfg.Port = port
		}
	}
}

// applyDefaultSettings applies default values for various configuration sections.
func applyDefaultSettings(cfg *Config, originalData []byte) {
	applyDefaultPortAndTimeout(cfg)
	applyDefaultServerSettings(cfg)
	applyDefaultCompressionSettings(cfg, originalData)
	applyDefaultJobsSettings(cfg)
	applyDefaultSchedulerSettings(cfg)
	applyDefaultEmailSettings(cfg)
	applyDefaultTypeScriptSettings(cfg)
}

// applyDefaultPortAndTimeout sets default values for port and timeout.
func applyDefaultPortAndTimeout(cfg *Config) {
	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30
	}
}

// applyDefaultServerSettings sets default server configuration values.
func applyDefaultServerSettings(cfg *Config) {
	if cfg.Server.PoolSize == 0 {
		cfg.Server.PoolSize = 10
	}
	if cfg.Server.ProfilerPort == "" {
		cfg.Server.ProfilerPort = "6060"
	}
	if cfg.Server.PerformanceInterval == 0 {
		cfg.Server.PerformanceInterval = 30
	}
	if cfg.Server.HTTPTimeout == 0 {
		cfg.Server.HTTPTimeout = 30
	}
	if cfg.Server.InitialCleanupDelay == 0 {
		cfg.Server.InitialCleanupDelay = 30
	}
}

// applyDefaultCompressionSettings sets default compression configuration.
func applyDefaultCompressionSettings(cfg *Config, originalData []byte) {
	if cfg.Compression.MinSize == 0 {
		cfg.Compression.MinSize = 1024 // Default: 1KB minimum size
	}
	if cfg.Compression.Level == 0 {
		cfg.Compression.Level = 6 // Default: balanced compression level
	}
	// Default to enabled if not explicitly set (nil/false in YAML becomes false in Go)
	// We check the YAML content to see if compression was explicitly disabled
	if !strings.Contains(string(originalData), "compression:") {
		cfg.Compression.Enabled = true // Default: enabled
	}
}

// applyDefaultJobsSettings sets default jobs configuration.
func applyDefaultJobsSettings(cfg *Config) {
	if cfg.Jobs.MaxWorkers == 0 {
		cfg.Jobs.MaxWorkers = 10
	}
	if cfg.Jobs.RetryAttempts == 0 {
		cfg.Jobs.RetryAttempts = 3
	}
	if cfg.Jobs.RetryDelay == 0 {
		cfg.Jobs.RetryDelay = 5
	}
	if cfg.Jobs.Timeout == 0 {
		cfg.Jobs.Timeout = 30
	}
	if cfg.Jobs.QueueSize == 0 {
		cfg.Jobs.QueueSize = 1000
	}
}

// applyDefaultSchedulerSettings sets default scheduler configuration.
func applyDefaultSchedulerSettings(cfg *Config) {
	// Default to enabled if not explicitly configured
	// cfg.Scheduler.Enabled defaults to false (Go zero value), which we want to override to true
	// Only set to true if no scheduler config was provided or if enabled wasn't explicitly set to false
	if cfg.Scheduler.Timezone == "" {
		cfg.Scheduler.Timezone = "UTC"
	}
	if cfg.Scheduler.Path == "" {
		cfg.Scheduler.Path = "./app/schedulers"
	}
	if cfg.Scheduler.LogLevel == "" {
		cfg.Scheduler.LogLevel = "info"
	}

	// Set enabled to true by default if tasks are configured or if it's not explicitly set
	if len(cfg.Scheduler.Tasks) > 0 {
		cfg.Scheduler.Enabled = true
	}

	// Apply defaults to individual tasks
	for i := range cfg.Scheduler.Tasks {
		task := &cfg.Scheduler.Tasks[i]
		if task.Timezone == "" {
			task.Timezone = cfg.Scheduler.Timezone
		}
		if task.Timeout <= 0 {
			task.Timeout = cfg.Timeout // Use global timeout
		}
		// Default enabled to true for individual tasks
		if task.Cron != "" && task.Handler != "" {
			// Only set enabled if cron and handler are provided
			// Keep the YAML value if explicitly set
		}
	}
}

// applyDefaultEmailSettings sets default email configuration.
func applyDefaultEmailSettings(cfg *Config) {
	if cfg.Email.DefaultDriver == "" {
		cfg.Email.DefaultDriver = "smtp"
	}
	if cfg.Email.FromAddress == "" {
		cfg.Email.FromAddress = "noreply@turboscript.dev"
	}
	if cfg.Email.FromName == "" {
		cfg.Email.FromName = "TurboScript"
	}
}

// applyDefaultTypeScriptSettings sets default TypeScript compiler configuration.
func applyDefaultTypeScriptSettings(cfg *Config) {
	// Set default external modules if not configured
	if len(cfg.TypeScript.ExternalModules) == 0 {
		cfg.TypeScript.ExternalModules = []string{
			"bcryptjs", "crypto", "node:crypto", "node:url", "node:assert", "node:util",
			"fs", "path", "os", "argon2",
		}
	}

	// Set default compilation target
	if cfg.TypeScript.Target == "" {
		cfg.TypeScript.Target = "ES2020"
	}

	// Set default output format
	if cfg.TypeScript.Format == "" {
		cfg.TypeScript.Format = "CommonJS"
	}

	// MinifyJS and SourceMaps default to false (no change needed for bool zero values)
}

// UnmarshalYAML implements custom YAML unmarshaling for EndpointConfig
// This allows the folder field to be either a boolean or a FolderConfig object.
func (e *EndpointConfig) UnmarshalYAML(unmarshal func(any) error) error {
	// Define a temporary struct with all fields
	type TempEndpointConfig struct {
		Route     string         `yaml:"route"`
		Method    string         `yaml:"method"`
		Path      string         `yaml:"path"`
		Table     string         `yaml:"table"`
		Operation string         `yaml:"operation"`
		Timeout   int            `yaml:"timeout"`
		Folder    any            `yaml:"folder"`
		Type      string         `yaml:"type"`
		Options   map[string]any `yaml:"options"`
		Index     string         `yaml:"index"`
		Layout    string         `yaml:"layout"`
		Markdown  bool           `yaml:"markdown"`
	}

	var temp TempEndpointConfig
	if err := unmarshal(&temp); err != nil {
		return err
	}

	// Copy simple fields
	e.Route = temp.Route
	e.Method = temp.Method
	e.Path = temp.Path
	e.Table = temp.Table
	e.Operation = temp.Operation
	e.Timeout = temp.Timeout
	e.Type = temp.Type
	e.Options = temp.Options
	e.Index = temp.Index
	e.Layout = temp.Layout
	e.Markdown = temp.Markdown

	// Handle the flexible folder field
	if err := e.handleFolderField(temp.Folder); err != nil {
		return err
	}

	return nil
}

// handleFolderField processes the flexible folder field configuration.
func (e *EndpointConfig) handleFolderField(folderValue any) error {
	if folderValue == nil {
		return nil
	}

	switch v := folderValue.(type) {
	case bool:
		// Simple boolean case: folder: true
		if v {
			e.FolderConfig = &FolderConfig{
				Type:   e.Type,
				Index:  e.Index,
				Layout: e.Layout,
			}
		}
	case map[string]any:
		// Nested configuration case (YAML maps are string keys)
		folderConfig := &FolderConfig{}

		// Convert and validate the folder configuration
		folderYAML, err := yaml.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal folder config: %w", err)
		}

		if err := yaml.Unmarshal(folderYAML, folderConfig); err != nil {
			return fmt.Errorf("failed to unmarshal folder config: %w", err)
		}

		// Provide fallback values from the main endpoint config
		if folderConfig.Type == "" {
			folderConfig.Type = e.Type
		}
		if folderConfig.Index == "" {
			folderConfig.Index = e.Index
		}
		if folderConfig.Layout == "" {
			folderConfig.Layout = e.Layout
		}

		e.FolderConfig = folderConfig
	default:
		return fmt.Errorf("folder field must be a boolean or object")
	}

	return nil
}

// IsAPIEndpoint returns true if this endpoint is a standard API endpoint (default type).
func (e *EndpointConfig) IsAPIEndpoint() bool {
	return e.GetType() == "api"
}

// IsWebSocketEndpoint returns true if this endpoint is a WebSocket endpoint.
func (e *EndpointConfig) IsWebSocketEndpoint() bool {
	return e.GetType() == "websocket" || e.Method == "WebSocket"
}

// IsSSEEndpoint returns true if this endpoint is a Server-Sent Events endpoint.
func (e *EndpointConfig) IsSSEEndpoint() bool {
	return e.GetType() == "sse" || e.Method == "SSE"
}

// IsRealtimeEndpoint returns true if this endpoint supports real-time communication.
func (e *EndpointConfig) IsRealtimeEndpoint() bool {
	return e.IsWebSocketEndpoint() || e.IsSSEEndpoint()
}

// IsFolderEndpoint returns true if this endpoint serves folder content (backward compatibility).
func (e *EndpointConfig) IsFolderEndpoint() bool {
	// Check new type field first
	endpointType := e.GetType()
	if endpointType == "markdown-html" || endpointType == "hybrid" {
		return true
	}
	// Backward compatibility: check legacy FolderConfig
	return e.FolderConfig != nil
}

// GetType returns the endpoint type, defaulting to "api" if not specified.
func (e *EndpointConfig) GetType() string {
	if e.Type != "" {
		return e.Type
	}
	// Auto-detect type from method for real-time endpoints
	if e.Method == "WebSocket" {
		return "websocket"
	}
	if e.Method == "SSE" {
		return "sse"
	}
	// Backward compatibility: if FolderConfig exists, infer type
	if e.FolderConfig != nil && e.FolderConfig.Type != "" {
		return e.FolderConfig.Type
	}
	// Default to API endpoint
	return "api"
}

// GetOption returns a specific option value from the Options map.
func (e *EndpointConfig) GetOption(key string) (any, bool) {
	if e.Options == nil {
		return nil, false
	}
	value, exists := e.Options[key]
	return value, exists
}

// GetOptionString returns a string option value with a default fallback.
func (e *EndpointConfig) GetOptionString(key, defaultValue string) string {
	if value, exists := e.GetOption(key); exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetIndexFile returns the index file for folder-based endpoints.
func (e *EndpointConfig) GetIndexFile() string {
	// Check new options first
	if indexFile := e.GetOptionString("index", ""); indexFile != "" {
		return indexFile
	}
	// Backward compatibility
	if e.FolderConfig != nil && e.FolderConfig.Index != "" {
		return e.FolderConfig.Index
	}
	return e.Index
}

// GetLayoutFile returns the layout file for folder-based endpoints.
func (e *EndpointConfig) GetLayoutFile() string {
	// Check new options first
	if layoutFile := e.GetOptionString("layout", ""); layoutFile != "" {
		return layoutFile
	}
	// Backward compatibility
	if e.FolderConfig != nil && e.FolderConfig.Layout != "" {
		return e.FolderConfig.Layout
	}
	return e.Layout
}

// IsMarkdownEnabled returns true if this endpoint should process markdown files.
func (e *EndpointConfig) IsMarkdownEnabled() bool {
	// Check new type field first
	if e.GetType() == "markdown-html" {
		return true
	}
	return e.Markdown
}

// GetEnvironmentVariables returns a merged map of environment variables.
//
// Priority order (highest to lowest):
//  1. System environment variables
//  2. Environment variables from config file
//
// This ensures that system environment variables can override config file values,
// which is important for secure production deployments.
func (c *Config) GetEnvironmentVariables() map[string]string {
	envVars := make(map[string]string)

	// Start with config file environment variables
	for key, value := range c.Env {
		envVars[key] = value
	}

	// Override with system environment variables if they exist
	for key := range c.Env {
		if sysValue := os.Getenv(key); sysValue != "" {
			envVars[key] = sysValue
		}
	}

	// Also add any additional system environment variables that might be relevant
	// This allows for runtime addition of environment variables without config changes
	relevantEnvKeys := []string{
		"APP_ENV",
		"JWT_ACCESS_SECRET",
		"JWT_REFRESH_SECRET",
	}

	for _, key := range relevantEnvKeys {
		if sysValue := os.Getenv(key); sysValue != "" {
			envVars[key] = sysValue
		}
	}

	return envVars
}

// GetEffectiveTimeout returns the effective timeout for an endpoint.
// If the endpoint has a specific timeout configured, it returns that value.
// Otherwise, it returns the global timeout from the configuration.
func (e *EndpointConfig) GetEffectiveTimeout(globalTimeout int) int {
	if e.Timeout > 0 {
		return e.Timeout
	}
	return globalTimeout
}

// GetEndpointTimeout returns the timeout in seconds for a specific endpoint.
// This is a convenience method that combines endpoint-specific and global timeout logic.
func (c *Config) GetEndpointTimeout(endpoint *EndpointConfig) int {
	if endpoint != nil {
		return endpoint.GetEffectiveTimeout(c.Timeout)
	}
	return c.Timeout
}

// GetDatabaseConnectionStringByName returns the connection string for a specific database connection.
func (c *Config) GetDatabaseConnectionStringByName(name string) (string, error) {
	conn, err := c.Database.getConnection(name)
	if err != nil {
		return "", err
	}

	return conn.BuildConnectionString()
}

// resolveStructEnvVariables recursively resolves environment variables for all string fields in a struct.
// This function uses reflection to automatically handle any struct type, ensuring comprehensive
// environment variable resolution across the entire configuration.
func resolveStructEnvVariables(v any) error {
	return resolveStructEnvVariablesRecursive(reflect.ValueOf(v).Elem())
}

// resolveStructEnvVariablesRecursive is the recursive implementation of resolveStructEnvVariables.
func resolveStructEnvVariablesRecursive(val reflect.Value) error {
	switch val.Kind() {
	case reflect.String:
		return resolveStringField(val)
	case reflect.Struct:
		return resolveStructFields(val)
	case reflect.Ptr:
		return resolvePointer(val)
	case reflect.Slice, reflect.Array:
		return resolveSliceOrArray(val)
	case reflect.Map:
		return resolveMap(val)
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer:
		// These types don't need environment variable resolution
		return nil
	}
	return nil
}

func resolveStringField(val reflect.Value) error {
	if val.CanSet() {
		originalValue := val.String()
		resolvedValue := resolveEnvVariables(originalValue)
		val.SetString(resolvedValue)
	}
	return nil
}

func resolveStructFields(val reflect.Value) error {
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.CanSet() {
			if err := resolveStructEnvVariablesRecursive(field); err != nil {
				return err
			}
		}
	}
	return nil
}

func resolvePointer(val reflect.Value) error {
	if !val.IsNil() && val.Elem().Kind() == reflect.Struct {
		return resolveStructEnvVariablesRecursive(val.Elem())
	}
	return nil
}

func resolveSliceOrArray(val reflect.Value) error {
	for i := 0; i < val.Len(); i++ {
		element := val.Index(i)
		if err := resolveStructEnvVariablesRecursive(element); err != nil {
			return err
		}
	}
	return nil
}

func resolveMap(val reflect.Value) error {
	for _, key := range val.MapKeys() {
		mapValue := val.MapIndex(key)
		switch mapValue.Kind() {
		case reflect.String:
			originalValue := mapValue.String()
			resolvedValue := resolveEnvVariables(originalValue)
			val.SetMapIndex(key, reflect.ValueOf(resolvedValue))
		case reflect.Struct, reflect.Ptr:
			if err := resolveStructEnvVariablesRecursive(mapValue); err != nil {
				return err
			}
		case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
			reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128,
			reflect.Array, reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.UnsafePointer:
			// No action needed for these types
		}
	}
	return nil
}
