package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvironmentVariableResolution(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		envVars     map[string]string
		expectError bool
		checkFields func(*Config) bool
	}{
		{
			name: "env vars with defaults in cache config",
			configYAML: `
cache:
  default: "redis-test"
  drivers:
    redis-test:
      driver: "redis"
      host: ${env:REDIS_HOST, "localhost"}
      port: ${env:REDIS_PORT, 6379}
      password: ${env:REDIS_PASSWORD, ""}
      db: ${env:REDIS_DB, 0}
      max_idle_connections: ${env:REDIS_MAX_IDLE, 10}
      max_active_connections: ${env:REDIS_MAX_ACTIVE, 50}
      idle_timeout: ${env:REDIS_IDLE_TIMEOUT, 300}
      read_timeout: ${env:REDIS_READ_TIMEOUT, 30}
      write_timeout: ${env:REDIS_WRITE_TIMEOUT, 30}
endpoints: []
`,
			envVars: map[string]string{
				"REDIS_HOST":     "redis.example.com",
				"REDIS_PORT":     "6380",
				"REDIS_PASSWORD": "secret123",
			},
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["redis-test"]
				return driver.Host == "redis.example.com" &&
					driver.Port == 6380 &&
					driver.Password == "secret123" &&
					driver.DB == 0 && // Should use default
					driver.MaxIdleConnections == 10 && // Should use default
					driver.MaxActiveConnections == 50 && // Should use default
					driver.IdleTimeout == 300 && // Should use default
					driver.ReadTimeout == 30 && // Should use default
					driver.WriteTimeout == 30 // Should use default
			},
		},
		{
			name: "env vars without defaults using env values",
			configYAML: `
cache:
  default: "memcached-test"
  drivers:
    memcached-test:
      driver: "memcached"
      host: ${env:MEMCACHED_HOST}
      port: ${env:MEMCACHED_PORT}
      max_idle_connections: ${env:MEMCACHED_MAX_IDLE}
      max_active_connections: ${env:MEMCACHED_MAX_ACTIVE}
      idle_timeout: ${env:MEMCACHED_IDLE_TIMEOUT}
      read_timeout: ${env:MEMCACHED_READ_TIMEOUT}
      write_timeout: ${env:MEMCACHED_WRITE_TIMEOUT}
endpoints: []
`,
			envVars: map[string]string{
				"MEMCACHED_HOST":          "memcached.example.com",
				"MEMCACHED_PORT":          "11212",
				"MEMCACHED_MAX_IDLE":      "5",
				"MEMCACHED_MAX_ACTIVE":    "25",
				"MEMCACHED_IDLE_TIMEOUT":  "600",
				"MEMCACHED_READ_TIMEOUT":  "15",
				"MEMCACHED_WRITE_TIMEOUT": "15",
			},
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["memcached-test"]
				return driver.Host == "memcached.example.com" &&
					driver.Port == 11212 &&
					driver.MaxIdleConnections == 5 &&
					driver.MaxActiveConnections == 25 &&
					driver.IdleTimeout == 600 &&
					driver.ReadTimeout == 15 &&
					driver.WriteTimeout == 15
			},
		},
		{
			name: "env vars without defaults and no env values (empty strings)",
			configYAML: `
cache:
  default: "file-test"
  drivers:
    file-test:
      driver: "file"
      root: ${env:FILE_CACHE_ROOT}
      max_size: ${env:FILE_CACHE_MAX_SIZE}
endpoints: []
`,
			envVars:     map[string]string{}, // No env vars set
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["file-test"]
				return driver.Root == "" && // Should be empty string
					driver.MaxSize == 0 // Should be 0 (empty string parsed as 0)
			},
		},
		{
			name: "database env vars with defaults",
			configYAML: `
database:
  default: "main"
  connections:
    main:
      driver: "postgres"
      host: ${env:DB_HOST, "localhost"}
      port: ${env:DB_PORT, 5432}
      username: ${env:DB_USERNAME, "turboscript_user"}
      password: ${env:DB_PASSWORD, "turboscript_pass"}
      database: ${env:DB_NAME, "turboscript"}
      max_open_connections: ${env:DB_MAX_OPEN, 10}
      max_idle_connections: ${env:DB_MAX_IDLE, 5}
      connection_timeout: ${env:DB_CONN_TIMEOUT, 30}
      max_lifetime: ${env:DB_MAX_LIFETIME, 60}
      max_idle_time: ${env:DB_MAX_IDLE_TIME, 30}
      ssl_mode: ${env:DB_SSL_MODE, "disable"}
      timezone: ${env:DB_TIMEZONE, "UTC"}
endpoints: []
`,
			envVars: map[string]string{
				"DB_HOST":          "postgres.example.com",
				"DB_PORT":          "5433",
				"DB_MAX_OPEN":      "20",
				"DB_MAX_IDLE":      "10",
				"DB_CONN_TIMEOUT":  "45",
				"DB_MAX_LIFETIME":  "120",
				"DB_MAX_IDLE_TIME": "60",
			},
			expectError: false,
			checkFields: func(cfg *Config) bool {
				conn := cfg.Database.Connections["main"]
				return conn.Host == "postgres.example.com" &&
					conn.Port == 5433 &&
					conn.Username == "turboscript_user" && // Default
					conn.Password == "turboscript_pass" && // Default
					conn.Database == "turboscript" && // Default
					conn.MaxOpenConnections == 20 &&
					conn.MaxIdleConnections == 10 &&
					conn.ConnectionTimeout == 45 &&
					conn.MaxLifetime == 120 &&
					conn.MaxIdleTime == 60 &&
					conn.SSLMode == "disable" && // Default
					conn.Timezone == "UTC" // Default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			// Create temporary config file
			tempDir, err := os.MkdirTemp("", "config_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "test_config.yml")
			err = os.WriteFile(configPath, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Test LoadConfig
			cfg, err := LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Errorf("Expected config but got nil")
				return
			}

			if tt.checkFields != nil && !tt.checkFields(cfg) {
				t.Errorf("Config fields validation failed")
			}
		})
	}
}

func TestCacheDriverConfigurations(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		checkFields func(*Config) bool
	}{
		{
			name: "memory cache driver",
			configYAML: `
cache:
  default: "memory-local"
  drivers:
    memory-local:
      driver: "memory"
      max_size: 100
      expiration: 3600
endpoints: []
`,
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["memory-local"]
				return driver.Driver == "memory" &&
					driver.MaxSize == 100 &&
					driver.Expiration == 3600
			},
		},
		{
			name: "redis cache driver",
			configYAML: `
cache:
  default: "redis-server"
  drivers:
    redis-server:
      driver: "redis"
      host: "localhost"
      port: 6379
      password: ""
      db: 0
      max_idle_connections: 10
      max_active_connections: 50
      idle_timeout: 300
      read_timeout: 30
      write_timeout: 30
endpoints: []
`,
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["redis-server"]
				return driver.Driver == "redis" &&
					driver.Host == "localhost" &&
					driver.Port == 6379 &&
					driver.Password == "" &&
					driver.DB == 0 &&
					driver.MaxIdleConnections == 10 &&
					driver.MaxActiveConnections == 50 &&
					driver.IdleTimeout == 300 &&
					driver.ReadTimeout == 30 &&
					driver.WriteTimeout == 30
			},
		},
		{
			name: "memcached cache driver",
			configYAML: `
cache:
  default: "memcached-server"
  drivers:
    memcached-server:
      driver: "memcached"
      host: "localhost"
      port: 11211
      max_idle_connections: 10
      max_active_connections: 50
      idle_timeout: 300
      read_timeout: 30
      write_timeout: 30
endpoints: []
`,
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["memcached-server"]
				return driver.Driver == "memcached" &&
					driver.Host == "localhost" &&
					driver.Port == 11211 &&
					driver.MaxIdleConnections == 10 &&
					driver.MaxActiveConnections == 50 &&
					driver.IdleTimeout == 300 &&
					driver.ReadTimeout == 30 &&
					driver.WriteTimeout == 30
			},
		},
		{
			name: "file cache driver",
			configYAML: `
cache:
  default: "file-system"
  drivers:
    file-system:
      driver: "file"
      root: "./cache"
      max_size: 10
endpoints: []
`,
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["file-system"]
				return driver.Driver == "file" &&
					driver.Root == "./cache" &&
					driver.MaxSize == 10
			},
		},
		{
			name: "multiple cache drivers",
			configYAML: `
cache:
  default: "memory-local"
  drivers:
    memory-local:
      driver: "memory"
      max_size: 100
      expiration: 3600
    redis-server:
      driver: "redis"
      host: "localhost"
      port: 6379
      password: ""
      db: 0
    memcached-server:
      driver: "memcached"
      host: "localhost"
      port: 11211
    file-system:
      driver: "file"
      root: "./cache"
      max_size: 10
endpoints: []
`,
			expectError: false,
			checkFields: func(cfg *Config) bool {
				if len(cfg.Cache.Drivers) != 4 {
					return false
				}

				memory := cfg.Cache.Drivers["memory-local"]
				redis := cfg.Cache.Drivers["redis-server"]
				memcached := cfg.Cache.Drivers["memcached-server"]
				file := cfg.Cache.Drivers["file-system"]

				return memory.Driver == "memory" &&
					redis.Driver == "redis" &&
					memcached.Driver == "memcached" &&
					file.Driver == "file" &&
					cfg.Cache.Default == "memory-local"
			},
		},
		{
			name: "cache drivers with environment variables",
			configYAML: `
cache:
  default: "redis-env"
  drivers:
    memory-local:
      driver: "memory"
      max_size: ${env:MEMORY_MAX_SIZE, 100}
      expiration: ${env:MEMORY_EXPIRATION, 3600}
    redis-env:
      driver: "redis"
      host: ${env:REDIS_HOST, "localhost"}
      port: ${env:REDIS_PORT, 6379}
      password: ${env:REDIS_PASSWORD, ""}
      db: ${env:REDIS_DB, 0}
      max_idle_connections: ${env:REDIS_MAX_IDLE, 10}
      max_active_connections: ${env:REDIS_MAX_ACTIVE, 50}
      idle_timeout: ${env:REDIS_IDLE_TIMEOUT, 300}
      read_timeout: ${env:REDIS_READ_TIMEOUT, 30}
      write_timeout: ${env:REDIS_WRITE_TIMEOUT, 30}
    memcached-env:
      driver: "memcached"
      host: ${env:MEMCACHED_HOST, "localhost"}
      port: ${env:MEMCACHED_PORT, 11211}
      max_idle_connections: ${env:MEMCACHED_MAX_IDLE, 10}
      max_active_connections: ${env:MEMCACHED_MAX_ACTIVE, 50}
      idle_timeout: ${env:MEMCACHED_IDLE_TIMEOUT, 300}
      read_timeout: ${env:MEMCACHED_READ_TIMEOUT, 30}
      write_timeout: ${env:MEMCACHED_WRITE_TIMEOUT, 30}
    file-env:
      driver: "file"
      root: ${env:FILE_CACHE_ROOT, "./cache"}
      max_size: ${env:FILE_CACHE_MAX_SIZE, 10}
endpoints: []
`,
			expectError: false,
			checkFields: func(cfg *Config) bool {
				if len(cfg.Cache.Drivers) != 4 {
					return false
				}

				memory := cfg.Cache.Drivers["memory-local"]
				redis := cfg.Cache.Drivers["redis-env"]
				memcached := cfg.Cache.Drivers["memcached-env"]
				file := cfg.Cache.Drivers["file-env"]

				return memory.Driver == "memory" &&
					memory.MaxSize == 100 &&
					memory.Expiration == 3600 &&
					redis.Driver == "redis" &&
					redis.Host == "localhost" &&
					redis.Port == 6379 &&
					redis.Password == "" &&
					redis.DB == 0 &&
					redis.MaxIdleConnections == 10 &&
					redis.MaxActiveConnections == 50 &&
					redis.IdleTimeout == 300 &&
					redis.ReadTimeout == 30 &&
					redis.WriteTimeout == 30 &&
					memcached.Driver == "memcached" &&
					memcached.Host == "localhost" &&
					memcached.Port == 11211 &&
					memcached.MaxIdleConnections == 10 &&
					memcached.MaxActiveConnections == 50 &&
					memcached.IdleTimeout == 300 &&
					memcached.ReadTimeout == 30 &&
					memcached.WriteTimeout == 30 &&
					file.Driver == "file" &&
					file.Root == "./cache" &&
					file.MaxSize == 10 &&
					cfg.Cache.Default == "redis-env"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tempDir, err := os.MkdirTemp("", "config_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "test_config.yml")
			err = os.WriteFile(configPath, []byte(tt.configYAML), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			// Test LoadConfig
			cfg, err := LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if cfg == nil {
				t.Errorf("Expected config but got nil")
				return
			}

			if tt.checkFields != nil && !tt.checkFields(cfg) {
				t.Errorf("Config fields validation failed")
			}
		})
	}
}
