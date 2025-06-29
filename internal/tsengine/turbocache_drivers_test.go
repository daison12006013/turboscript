package tsengine

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
)

// getRedisHost returns the appropriate Redis host for testing environment
func getRedisHost() string {
	if host := os.Getenv("REDIS_HOST"); host != "" {
		return host
	}
	// Check if we're running in Docker environment
	if os.Getenv("DOCKER_ENV") == "true" {
		return "redis"
	}
	// Default to localhost for local development
	return "localhost"
}

// getMemcachedHost returns the appropriate Memcached host for testing environment
func getMemcachedHost() string {
	if host := os.Getenv("MEMCACHED_HOST"); host != "" {
		return host
	}
	// Check if we're running in Docker environment
	if os.Getenv("DOCKER_ENV") == "true" {
		return "memcached"
	}
	// Default to localhost for local development
	return "localhost"
}

func TestTurboCacheAllDrivers_BasicOperations(t *testing.T) {
	// Create test configuration with all drivers
	cacheConfig := &config.CacheConfig{
		Default: "memory-test",
		Drivers: map[string]config.CacheDriverConfig{
			"memory-test": {
				Driver: "memory",
			},
			"redis-test": {
				Driver:   "redis",
				Host:     getRedisHost(),
				Port:     6379,
				Password: "turboscript_redis_pass",
				DB:       1, // Use DB 1 for tests
			},
			"memcached-test": {
				Driver: "memcached",
				Host:   getMemcachedHost(),
				Port:   11211,
			},
			"file-test": {
				Driver: "file",
				Root:   "./test_cache",
			},
		},
	}

	// Create cache utils
	cacheUtils := NewTurboCacheUtils(cacheConfig)

	// Test data
	testKey := "test:all-drivers"
	testValue := map[string]any{
		"driver":    "test",
		"timestamp": time.Now().Unix(),
		"data":      "test_value",
		"nested": map[string]any{
			"level": 2,
			"value": "nested_test",
		},
	}
	testTTL := int64(60)

	drivers := []string{"memory-test", "redis-test", "memcached-test", "file-test"}

	for _, driverName := range drivers {
		t.Run(driverName, func(t *testing.T) {
			driver := cacheUtils.getDriver(driverName)
			if driver == nil {
				t.Skipf("Driver %s not available", driverName)
				return
			}

			// Test connection first for external services
			if driverName == "redis-test" || driverName == "memcached-test" {
				// Try a simple set/get to verify connection
				driver.set("test:connection", "test", 10)
				if _, exists := driver.get("test:connection"); !exists {
					t.Skipf("Driver %s connection test failed - service may not be available", driverName)
					return
				}
				driver.del("test:connection")
			}

			// Clean up before test
			driver.del(testKey)

			// Test set operation
			driver.set(testKey, testValue, testTTL)

			// Test get operation
			retrieved, exists := driver.get(testKey)
			if !exists {
				t.Errorf("Key should exist after set operation")
				return
			}

			// Test has operation
			if !driver.has(testKey) {
				t.Errorf("has() should return true for existing key")
			}

			// Verify the retrieved value matches the original
			switch driverName {
			case "memory-test":
				// Memory driver should return the exact object
				if !compareValues(retrieved, testValue) {
					t.Errorf("Memory driver: retrieved value doesn't match original.\nExpected: %+v\nGot: %+v", testValue, retrieved)
				}
			default:
				// Other drivers use JSON serialization, so compare JSON strings
				expectedJSON, _ := json.Marshal(testValue)
				retrievedJSON, _ := json.Marshal(retrieved)
				if string(expectedJSON) != string(retrievedJSON) {
					t.Errorf("%s driver: JSON serialization doesn't match.\nExpected: %s\nGot: %s", driverName, string(expectedJSON), string(retrievedJSON))
				}
			}

			// Test del operation
			driver.del(testKey)
			_, exists = driver.get(testKey)
			if exists {
				t.Errorf("Key should not exist after delete operation")
			}

			// Test has operation after delete
			if driver.has(testKey) {
				t.Errorf("has() should return false for deleted key")
			}
		})
	}

	// Clean up test cache directory
	_ = os.RemoveAll("./test_cache")
}

func TestTurboCacheAllDrivers_DataTypes(t *testing.T) {
	// Test different data types
	cacheConfig := &config.CacheConfig{
		Default: "memory-test",
		Drivers: map[string]config.CacheDriverConfig{
			"memory-test": {
				Driver: "memory",
			},
			"redis-test": {
				Driver:   "redis",
				Host:     getRedisHost(),
				Port:     6379,
				Password: "turboscript_redis_pass",
				DB:       1,
			},
			"memcached-test": {
				Driver: "memcached",
				Host:   getMemcachedHost(),
				Port:   11211,
			},
			"file-test": {
				Driver: "file",
				Root:   "./test_cache_types",
			},
		},
	}

	cacheUtils := NewTurboCacheUtils(cacheConfig)

	testCases := []struct {
		name  string
		value any
	}{
		{"string", "hello world"},
		{"int", 42},
		{"float", 3.14159},
		{"bool", true},
		{"array", []string{"a", "b", "c"}},
		{"object", map[string]any{"key": "value", "number": 123}},
		{"complex", map[string]any{
			"string":  "test",
			"number":  456,
			"boolean": false,
			"array":   []any{1, 2, 3},
			"nested":  map[string]any{"deep": "value"},
		}},
	}

	drivers := []string{"memory-test", "redis-test", "memcached-test", "file-test"}

	for _, driver := range drivers {
		for _, tc := range testCases {
			t.Run(driver+"_"+tc.name, func(t *testing.T) {
				d := cacheUtils.getDriver(driver)
				if d == nil {
					t.Skipf("Driver %s not available", driver)
					return
				}

				// Test connection first for external services
				if driver == "redis-test" || driver == "memcached-test" {
					// Try a simple set/get to verify connection
					d.set("test:connection", "test", 10)
					if _, exists := d.get("test:connection"); !exists {
						t.Skipf("Driver %s connection test failed - service may not be available", driver)
						return
					}
					d.del("test:connection")
				}

				key := "test:" + tc.name
				d.del(key) // Clean up

				// Set and get the value
				d.set(key, tc.value, 60)
				retrieved, exists := d.get(key)

				if !exists {
					t.Errorf("Value should exist after set")
					return
				}

				// For memory driver, check exact match
				// For others, check JSON serialization equivalence
				if !compareValues(retrieved, tc.value) {
					t.Errorf("Value mismatch for driver %s.\nExpected: %+v\nGot: %+v", driver, tc.value, retrieved)
				}

				d.del(key) // Clean up
			})
		}
	}

	// Clean up test cache directory
	_ = os.RemoveAll("./test_cache_types")
}

func TestTurboCacheAllDrivers_TTL(t *testing.T) {
	// Test TTL functionality (only for drivers that support it)
	cacheConfig := &config.CacheConfig{
		Default: "memory-test",
		Drivers: map[string]config.CacheDriverConfig{
			"memory-test": {
				Driver: "memory",
			},
			"redis-test": {
				Driver:   "redis",
				Host:     getRedisHost(),
				Port:     6379,
				Password: "turboscript_redis_pass",
				DB:       1,
			},
			// Note: memcached and file drivers support TTL but testing with short TTL is difficult
		},
	}

	cacheUtils := NewTurboCacheUtils(cacheConfig)

	testKey := "test:ttl"
	testValue := "expires_soon"
	shortTTL := int64(1) // 1 second

	// Test drivers that support TTL
	ttlDrivers := []string{"memory-test", "redis-test"}

	for _, driverName := range ttlDrivers {
		t.Run(driverName+"_ttl", func(t *testing.T) {
			driver := cacheUtils.getDriver(driverName)
			if driver == nil {
				t.Skipf("Driver %s not available", driverName)
				return
			}

			// Test connection first for external services
			if driverName == "redis-test" {
				// Try a simple set/get to verify connection
				driver.set("test:connection", "test", 10)
				if _, exists := driver.get("test:connection"); !exists {
					t.Skipf("Driver %s connection test failed - service may not be available", driverName)
					return
				}
				driver.del("test:connection")
			}

			// Set with short TTL
			driver.set(testKey, testValue, shortTTL)

			// Should exist immediately
			if !driver.has(testKey) {
				t.Errorf("Key should exist immediately after set")
			}

			// Wait for expiration
			time.Sleep(2 * time.Second)

			// Should not exist after TTL
			if driver.has(testKey) {
				t.Errorf("Key should have expired after TTL")
			}

			_, exists := driver.get(testKey)
			if exists {
				t.Errorf("get() should return false for expired key")
			}
		})
	}
}

// Helper function to compare values for memory driver
func compareValues(a, b any) bool {
	switch valA := a.(type) {
	case bool:
		if valB, ok := b.(float64); ok {
			return (valA && valB == 1) || (!valA && valB == 0)
		}
	case float64:
		if valB, ok := b.(bool); ok {
			return (valB && valA == 1) || (!valB && valA == 0)
		}
	}

	aJSON, errA := json.Marshal(a)
	if errA != nil {
		return false
	}

	bJSON, errB := json.Marshal(b)
	if errB != nil {
		return false
	}

	return string(aJSON) == string(bJSON)
}
