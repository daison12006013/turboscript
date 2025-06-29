package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/redis/go-redis/v9"
)

// TestIntegrationCacheDriversWithDocker tests all cache drivers in a Docker environment
func TestIntegrationCacheDriversWithDocker(t *testing.T) {
	// Skip if not in Docker environment or if Redis/Memcached are not available
	if os.Getenv("DOCKER_ENV") != "true" {
		t.Skip("Skipping integration test - not in Docker environment")
	}

	tests := []struct {
		name           string
		configYAML     string
		envVars        map[string]string
		testConnection func(*testing.T, *Config) bool
	}{
		{
			name: "redis integration with docker services",
			configYAML: `
cache:
  default: "redis-docker"
  drivers:
    redis-docker:
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
				"REDIS_HOST":     "redis",
				"REDIS_PORT":     "6379",
				"REDIS_PASSWORD": "turboscript_redis_pass",
			},
			testConnection: func(t *testing.T, cfg *Config) bool {
				driver := cfg.Cache.Drivers["redis-docker"]
				if driver.Driver != "redis" {
					t.Errorf("Expected redis driver, got %s", driver.Driver)
					return false
				}

				// Test actual Redis connection
				client := redis.NewClient(&redis.Options{
					Addr:     fmt.Sprintf("%s:%d", driver.Host, driver.Port),
					Password: driver.Password,
					DB:       driver.DB,
				})
				defer client.Close()

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				_, err := client.Ping(ctx).Result()
				if err != nil {
					t.Logf("Redis connection failed: %v", err)
					return false
				}

				// Test set/get operations
				err = client.Set(ctx, "test:integration", "test_value", 10*time.Second).Err()
				if err != nil {
					t.Errorf("Failed to set Redis key: %v", err)
					return false
				}

				val, err := client.Get(ctx, "test:integration").Result()
				if err != nil {
					t.Errorf("Failed to get Redis key: %v", err)
					return false
				}

				if val != "test_value" {
					t.Errorf("Expected 'test_value', got '%s'", val)
					return false
				}

				t.Logf("Redis integration test passed: host=%s, port=%d, password=%s",
					driver.Host, driver.Port, driver.Password)
				return true
			},
		},
		{
			name: "memcached integration with docker services",
			configYAML: `
cache:
  default: "memcached-docker"
  drivers:
    memcached-docker:
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
				"MEMCACHED_HOST": "memcached",
				"MEMCACHED_PORT": "11211",
			},
			testConnection: func(t *testing.T, cfg *Config) bool {
				driver := cfg.Cache.Drivers["memcached-docker"]
				if driver.Driver != "memcached" {
					t.Errorf("Expected memcached driver, got %s", driver.Driver)
					return false
				}

				// Test actual Memcached connection
				client := memcache.New(fmt.Sprintf("%s:%d", driver.Host, driver.Port))
				client.Timeout = 5 * time.Second

				// Test set/get operations
				err := client.Set(&memcache.Item{
					Key:        "test:integration",
					Value:      []byte("test_value"),
					Expiration: 10,
				})
				if err != nil {
					t.Logf("Memcached connection/set failed: %v", err)
					return false
				}

				item, err := client.Get("test:integration")
				if err != nil {
					t.Errorf("Failed to get Memcached key: %v", err)
					return false
				}

				if string(item.Value) != "test_value" {
					t.Errorf("Expected 'test_value', got '%s'", string(item.Value))
					return false
				}

				t.Logf("Memcached integration test passed: host=%s, port=%d",
					driver.Host, driver.Port)
				return true
			},
		},
		{
			name: "mixed cache drivers with environment variables",
			configYAML: `
cache:
  default: "redis-docker"
  drivers:
    memory-local:
      driver: "memory"
      max_size: ${env:MEMORY_MAX_SIZE, 100}
      expiration: ${env:MEMORY_EXPIRATION, 3600}
    redis-docker:
      driver: "redis"
      host: ${env:REDIS_HOST, "localhost"}
      port: ${env:REDIS_PORT, 6379}
      password: ${env:REDIS_PASSWORD, ""}
      db: ${env:REDIS_DB, 0}
    memcached-docker:
      driver: "memcached"
      host: ${env:MEMCACHED_HOST, "localhost"}
      port: ${env:MEMCACHED_PORT, 11211}
    file-system:
      driver: "file"
      root: ${env:FILE_CACHE_ROOT, "./cache"}
      max_size: ${env:FILE_CACHE_MAX_SIZE, 10}
endpoints: []
`,
			envVars: map[string]string{
				"REDIS_HOST":        "redis",
				"REDIS_PORT":        "6379",
				"REDIS_PASSWORD":    "turboscript_redis_pass",
				"MEMCACHED_HOST":    "memcached",
				"MEMCACHED_PORT":    "11211",
				"FILE_CACHE_ROOT":   "./cache",
				"MEMORY_MAX_SIZE":   "200",
				"MEMORY_EXPIRATION": "7200",
			},
			testConnection: func(t *testing.T, cfg *Config) bool {
				// Check that all drivers are configured correctly
				drivers := []string{"memory-local", "redis-docker", "memcached-docker", "file-system"}
				if len(cfg.Cache.Drivers) != len(drivers) {
					t.Errorf("Expected %d drivers, got %d", len(drivers), len(cfg.Cache.Drivers))
					return false
				}

				for _, driverName := range drivers {
					if _, exists := cfg.Cache.Drivers[driverName]; !exists {
						t.Errorf("Driver %s not found in configuration", driverName)
						return false
					}
				}

				// Verify environment variable resolution
				memory := cfg.Cache.Drivers["memory-local"]
				if memory.MaxSize != 200 || memory.Expiration != 7200 {
					t.Errorf("Memory driver env vars not resolved correctly: maxSize=%d, expiration=%d",
						memory.MaxSize, memory.Expiration)
					return false
				}

				redis := cfg.Cache.Drivers["redis-docker"]
				if redis.Host != "redis" || redis.Port != 6379 || redis.Password != "turboscript_redis_pass" {
					t.Errorf("Redis driver env vars not resolved correctly: host=%s, port=%d, password=%s",
						redis.Host, redis.Port, redis.Password)
					return false
				}

				memcached := cfg.Cache.Drivers["memcached-docker"]
				if memcached.Host != "memcached" || memcached.Port != 11211 {
					t.Errorf("Memcached driver env vars not resolved correctly: host=%s, port=%d",
						memcached.Host, memcached.Port)
					return false
				}

				file := cfg.Cache.Drivers["file-system"]
				if file.Root != "./cache" || file.MaxSize != 10 {
					t.Errorf("File driver env vars not resolved correctly: root=%s, maxSize=%d",
						file.Root, file.MaxSize)
					return false
				}

				t.Log("All cache drivers configured correctly with environment variables")
				return true
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
			tempDir, err := os.MkdirTemp("", "config_integration_test")
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
			if err != nil {
				t.Fatalf("LoadConfig failed: %v", err)
			}

			if cfg == nil {
				t.Fatal("Expected config but got nil")
			}

			// Run connection test
			if !tt.testConnection(t, cfg) {
				t.Errorf("Integration test failed for %s", tt.name)
			}
		})
	}
}

// Helper function to check if services are available
func TestDockerServicesAvailability(t *testing.T) {
	if os.Getenv("DOCKER_ENV") != "true" {
		t.Skip("Skipping Docker services availability test - not in Docker environment")
	}

	t.Run("redis_availability", func(t *testing.T) {
		client := redis.NewClient(&redis.Options{
			Addr:     "redis:6379",
			Password: "turboscript_redis_pass",
			DB:       0,
		})
		defer client.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		result, err := client.Ping(ctx).Result()
		if err != nil {
			t.Errorf("Redis service not available: %v", err)
		} else {
			t.Logf("Redis service available: %s", result)
		}
	})

	t.Run("memcached_availability", func(t *testing.T) {
		client := memcache.New("memcached:11211")
		client.Timeout = 5 * time.Second

		err := client.Set(&memcache.Item{
			Key:        "test:availability",
			Value:      []byte("available"),
			Expiration: 10,
		})
		if err != nil {
			t.Errorf("Memcached service not available: %v", err)
		} else {
			t.Log("Memcached service available")
		}
	})
}
