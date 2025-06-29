package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDriversWithEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		envVars     map[string]string
		expectError bool
		checkFields func(*Config) bool
	}{
		{
			name: "redis cache driver with env vars",
			configYAML: `
cache:
  default: "redis-server"
  drivers:
    redis-server:
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
				"REDIS_DB":       "1",
			},
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["redis-server"]
				return driver.Driver == "redis" &&
					driver.Host == "redis.example.com" &&
					driver.Port == 6380 &&
					driver.Password == "secret123" &&
					driver.DB == 1 &&
					driver.MaxIdleConnections == 10 && // Should use default
					driver.MaxActiveConnections == 50 && // Should use default
					driver.IdleTimeout == 300 && // Should use default
					driver.ReadTimeout == 30 && // Should use default
					driver.WriteTimeout == 30 // Should use default
			},
		},
		{
			name: "memcached cache driver with env vars",
			configYAML: `
cache:
  default: "memcached-server"
  drivers:
    memcached-server:
      driver: "memcached"
      host: ${env:MEMCACHED_HOST, "localhost"}
      port: ${env:MEMCACHED_PORT, 11211}
      max_idle_connections: ${env:MEMCACHED_MAX_IDLE, 10}
      max_active_connections: ${env:MEMCACHED_MAX_ACTIVE, 50}
      idle_timeout: ${env:MEMCACHED_IDLE_TIMEOUT, 300}
      read_timeout: ${env:MEMCACHED_READ_TIMEOUT, 30}
      write_timeout: ${env:MEMCACHED_WRITE_TIMEOUT, 30}
endpoints: []
`,
			envVars: map[string]string{
				"MEMCACHED_HOST":     "memcached.example.com",
				"MEMCACHED_PORT":     "11212",
				"MEMCACHED_MAX_IDLE": "5",
			},
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["memcached-server"]
				return driver.Driver == "memcached" &&
					driver.Host == "memcached.example.com" &&
					driver.Port == 11212 &&
					driver.MaxIdleConnections == 5 &&
					driver.MaxActiveConnections == 50 && // Should use default
					driver.IdleTimeout == 300 && // Should use default
					driver.ReadTimeout == 30 && // Should use default
					driver.WriteTimeout == 30 // Should use default
			},
		},
		{
			name: "file cache driver with env vars",
			configYAML: `
cache:
  default: "file-system"
  drivers:
    file-system:
      driver: "file"
      root: ${env:FILE_CACHE_ROOT, "./cache"}
      max_size: ${env:FILE_CACHE_MAX_SIZE, 10}
endpoints: []
`,
			envVars: map[string]string{
				"FILE_CACHE_ROOT":     "/tmp/custom_cache",
				"FILE_CACHE_MAX_SIZE": "50",
			},
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["file-system"]
				return driver.Driver == "file" &&
					driver.Root == "/tmp/custom_cache" &&
					driver.MaxSize == 50
			},
		},
		{
			name: "memory cache driver with env vars",
			configYAML: `
cache:
  default: "memory-local"
  drivers:
    memory-local:
      driver: "memory"
      max_size: ${env:MEMORY_MAX_SIZE, 100}
      expiration: ${env:MEMORY_EXPIRATION, 3600}
endpoints: []
`,
			envVars: map[string]string{
				"MEMORY_MAX_SIZE":   "200",
				"MEMORY_EXPIRATION": "7200",
			},
			expectError: false,
			checkFields: func(cfg *Config) bool {
				driver := cfg.Cache.Drivers["memory-local"]
				return driver.Driver == "memory" &&
					driver.MaxSize == 200 &&
					driver.Expiration == 7200
			},
		},
		{
			name: "all cache drivers with env vars",
			configYAML: `
cache:
  default: "redis-server"
  drivers:
    memory-local:
      driver: "memory"
      max_size: ${env:MEMORY_MAX_SIZE, 100}
      expiration: ${env:MEMORY_EXPIRATION, 3600}
    redis-server:
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
    memcached-server:
      driver: "memcached"
      host: ${env:MEMCACHED_HOST, "localhost"}
      port: ${env:MEMCACHED_PORT, 11211}
      max_idle_connections: ${env:MEMCACHED_MAX_IDLE, 10}
      max_active_connections: ${env:MEMCACHED_MAX_ACTIVE, 50}
      idle_timeout: ${env:MEMCACHED_IDLE_TIMEOUT, 300}
      read_timeout: ${env:MEMCACHED_READ_TIMEOUT, 30}
      write_timeout: ${env:MEMCACHED_WRITE_TIMEOUT, 30}
    file-system:
      driver: "file"
      root: ${env:FILE_CACHE_ROOT, "./cache"}
      max_size: ${env:FILE_CACHE_MAX_SIZE, 10}
endpoints: []
`,
			envVars: map[string]string{
				"REDIS_HOST":      "redis",
				"REDIS_PORT":      "6379",
				"REDIS_PASSWORD":  "turboscript_redis_pass",
				"MEMCACHED_HOST":  "memcached",
				"MEMCACHED_PORT":  "11211",
				"FILE_CACHE_ROOT": "./cache",
			},
			expectError: false,
			checkFields: func(cfg *Config) bool {
				if len(cfg.Cache.Drivers) != 4 {
					return false
				}

				memory := cfg.Cache.Drivers["memory-local"]
				redis := cfg.Cache.Drivers["redis-server"]
				memcached := cfg.Cache.Drivers["memcached-server"]
				file := cfg.Cache.Drivers["file-system"]

				return cfg.Cache.Default == "redis-server" &&
					memory.Driver == "memory" &&
					memory.MaxSize == 100 && // default
					memory.Expiration == 3600 && // default
					redis.Driver == "redis" &&
					redis.Host == "redis" &&
					redis.Port == 6379 &&
					redis.Password == "turboscript_redis_pass" &&
					redis.DB == 0 && // default
					redis.MaxIdleConnections == 10 && // default
					redis.MaxActiveConnections == 50 && // default
					redis.IdleTimeout == 300 && // default
					redis.ReadTimeout == 30 && // default
					redis.WriteTimeout == 30 && // default
					memcached.Driver == "memcached" &&
					memcached.Host == "memcached" &&
					memcached.Port == 11211 &&
					memcached.MaxIdleConnections == 10 && // default
					memcached.MaxActiveConnections == 50 && // default
					memcached.IdleTimeout == 300 && // default
					memcached.ReadTimeout == 30 && // default
					memcached.WriteTimeout == 30 && // default
					file.Driver == "file" &&
					file.Root == "./cache" &&
					file.MaxSize == 10 // default
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

func TestResolveEnvVariablesRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "quoted string default",
			input:    `${env:TEST_DB_HOST_UNIQUE, "localhost"}`,
			envVars:  map[string]string{},
			expected: "localhost",
		},
		{
			name:     "unquoted numeric default",
			input:    `${env:REDIS_PORT, 6379}`,
			envVars:  map[string]string{},
			expected: "6379",
		},
		{
			name:     "unquoted boolean default",
			input:    `${env:DEBUG, true}`,
			envVars:  map[string]string{},
			expected: "true",
		},
		{
			name:     "env var overrides default",
			input:    `${env:REDIS_PORT, 6379}`,
			envVars:  map[string]string{"REDIS_PORT": "6380"},
			expected: "6380",
		},
		{
			name:     "env var without default",
			input:    `${env:API_KEY}`,
			envVars:  map[string]string{},
			expected: "",
		},
		{
			name:     "env var without default with value",
			input:    `${env:API_KEY}`,
			envVars:  map[string]string{"API_KEY": "secret123"},
			expected: "secret123",
		},
		{
			name:     "multiple env vars in one string",
			input:    `host: ${env:TEST_DB_HOST_UNIQUE, "localhost"}, port: ${env:TEST_DB_PORT_UNIQUE, 5432}`,
			envVars:  map[string]string{"TEST_DB_HOST_UNIQUE": "postgres.example.com"},
			expected: "host: postgres.example.com, port: 5432",
		},
		{
			name:     "whitespace handling",
			input:    `${env:TEST_PORT_UNIQUE, 8080 }`,
			envVars:  map[string]string{},
			expected: "8080",
		},
		{
			name:     "no env vars in string",
			input:    "plain text without env vars",
			envVars:  map[string]string{},
			expected: "plain text without env vars",
		},
		{
			name: "complex YAML-like structure",
			input: `driver: "redis"
host: ${env:REDIS_HOST, "localhost"}
port: ${env:REDIS_PORT, 6379}
password: ${env:REDIS_PASSWORD, ""}
db: ${env:REDIS_DB, 0}`,
			envVars: map[string]string{
				"REDIS_HOST":     "redis.example.com",
				"REDIS_PORT":     "6380",
				"REDIS_PASSWORD": "x",
			},
			expected: `driver: "redis"
host: redis.example.com
port: 6380
password: x
db: 0`,
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

			result := resolveEnvVariables(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
