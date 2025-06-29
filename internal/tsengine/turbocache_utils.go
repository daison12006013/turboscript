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

// Package tsengine provides shared utilities for cache handling.
package tsengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/dop251/goja"
	"github.com/redis/go-redis/v9"
)

// TurboCacheUtils provides shared turboCache functionality.

// turboCacheMemoryDriver is a simple in-memory cache with TTL support.
type turboCacheMemoryDriver struct {
	mu    sync.Mutex
	store map[string]turboCacheMemoryItem
}

type turboCacheMemoryItem struct {
	value     any
	expiresAt int64 // unix timestamp in seconds
}

func newTurboCacheMemoryDriver() *turboCacheMemoryDriver {
	return &turboCacheMemoryDriver{
		store: make(map[string]turboCacheMemoryItem),
	}
}

func (d *turboCacheMemoryDriver) get(key string) (any, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	item, ok := d.store[key]
	if !ok {
		return nil, false
	}
	if item.expiresAt > 0 && time.Now().Unix() > item.expiresAt {
		delete(d.store, key)
		return nil, false
	}
	return item.value, true
}

func (d *turboCacheMemoryDriver) set(key string, value any, ttl int64) {
	logger.Debug(">>> Setting cache key %s with value %v and ttl %d", key, value, ttl)
	var expires int64
	if ttl > 0 {
		expires = time.Now().Unix() + ttl
	}
	d.mu.Lock()
	d.store[key] = turboCacheMemoryItem{value: value, expiresAt: expires}
	d.mu.Unlock()
}

func (d *turboCacheMemoryDriver) del(key string) {
	d.mu.Lock()
	delete(d.store, key)
	d.mu.Unlock()
}

func (d *turboCacheMemoryDriver) has(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	item, ok := d.store[key]
	if !ok {
		return false
	}
	if item.expiresAt > 0 && time.Now().Unix() > item.expiresAt {
		delete(d.store, key)
		return false
	}
	return true
}

func (d *turboCacheMemoryDriver) flush() {
	d.mu.Lock()
	d.store = make(map[string]turboCacheMemoryItem)
	d.mu.Unlock()
}

// turboCacheRedisDriver wraps go-redis for basic cache operations.
type turboCacheRedisDriver struct {
	client *redis.Client
}

func newTurboCacheRedisDriverWithConfig(driverConfig config.CacheDriverConfig) *turboCacheRedisDriver {
	addr := fmt.Sprintf("%s:%d", driverConfig.Host, driverConfig.Port)
	logger.Debug("Creating Redis client with addr: %s, password: %s, DB: %d", addr, driverConfig.Password, driverConfig.DB)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: driverConfig.Password,
		DB:       driverConfig.DB,
	})

	// Test connection
	ctx := context.Background()
	if _, err := client.Ping(ctx).Result(); err != nil {
		logger.Error("Failed to connect to Redis at %s: %v", addr, err)
	} else {
		logger.Debug("Successfully connected to Redis at %s", addr)
	}

	return &turboCacheRedisDriver{client: client}
}

func (d *turboCacheRedisDriver) get(key string) (any, bool) {
	ctx := context.Background()
	val, err := d.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		logger.Debug("Redis get: key '%s' not found", key)
		return nil, false
	} else if err != nil {
		logger.Error("Redis get error for key '%s': %v", key, err)
		return nil, false
	}

	logger.Debug("Redis get: key '%s' found with value '%s'", key, val)
	// Try to deserialize JSON, fallback to string if it fails
	var result any
	if err := json.Unmarshal([]byte(val), &result); err == nil {
		return result, true
	}
	return val, true
}

func (d *turboCacheRedisDriver) set(key string, value any, ttl int64) {
	ctx := context.Background()
	var exp time.Duration
	if ttl > 0 {
		exp = time.Duration(ttl) * time.Second
	}

	// Serialize complex values to JSON
	var serializedValue any
	switch v := value.(type) {
	case string, int, int64, float64, bool:
		serializedValue = v
	default:
		if jsonBytes, err := json.Marshal(v); err == nil {
			serializedValue = string(jsonBytes)
		} else {
			serializedValue = fmt.Sprintf("%v", v)
		}
	}

	logger.Debug("Redis set: key '%s', value '%v', ttl '%v'", key, serializedValue, exp)
	err := d.client.Set(ctx, key, serializedValue, exp).Err()
	if err != nil {
		logger.Error("Redis set error for key '%s': %v", key, err)
	} else {
		logger.Debug("Redis set: successfully set key '%s'", key)
	}
}

func (d *turboCacheRedisDriver) del(key string) {
	ctx := context.Background()
	_ = d.client.Del(ctx, key).Err()
}

func (d *turboCacheRedisDriver) has(key string) bool {
	ctx := context.Background()
	n, err := d.client.Exists(ctx, key).Result()
	return err == nil && n > 0
}

func (d *turboCacheRedisDriver) flush() {
	ctx := context.Background()
	_ = d.client.FlushDB(ctx).Err()
}

// turboCacheMemcachedDriver wraps gomemcache for basic cache operations.
type turboCacheMemcachedDriver struct {
	client *memcache.Client
}

func newTurboCacheMemcachedDriver(host string, port int) *turboCacheMemcachedDriver {
	addr := fmt.Sprintf("%s:%d", host, port)
	logger.Debug("Creating Memcached client with addr: %s", addr)
	client := memcache.New(addr)

	// Test connection by setting a test value
	testItem := &memcache.Item{
		Key:   "test:connection",
		Value: []byte("test"),
	}
	if err := client.Set(testItem); err != nil {
		logger.Error("Failed to connect to Memcached at %s: %v", addr, err)
	} else {
		logger.Debug("Successfully connected to Memcached at %s", addr)
		// Clean up test item
		_ = client.Delete("test:connection")
	}

	return &turboCacheMemcachedDriver{client: client}
}

func (d *turboCacheMemcachedDriver) get(key string) (any, bool) {
	item, err := d.client.Get(key)
	if err != nil {
		return nil, false
	}

	// Try to deserialize JSON, fallback to string if it fails
	var result any
	if err := json.Unmarshal(item.Value, &result); err == nil {
		return result, true
	}
	return string(item.Value), true
}

func (d *turboCacheMemcachedDriver) set(key string, value any, ttl int64) {
	var exp int32
	if ttl > 0 {
		// Safe conversion with bounds checking to prevent integer overflow
		if ttl > 2147483647 { // max int32 value
			exp = 2147483647
		} else {
			exp = int32(ttl)
		}
	}

	// Serialize complex values to JSON
	var serializedValue []byte
	switch v := value.(type) {
	case string:
		serializedValue = []byte(v)
	case int, int64, float64, bool:
		serializedValue = []byte(fmt.Sprintf("%v", v))
	default:
		if jsonBytes, err := json.Marshal(v); err == nil {
			serializedValue = jsonBytes
		} else {
			serializedValue = []byte(fmt.Sprintf("%v", v))
		}
	}

	item := &memcache.Item{
		Key:        key,
		Value:      serializedValue,
		Expiration: exp,
	}
	_ = d.client.Set(item)
}

func (d *turboCacheMemcachedDriver) del(key string) {
	_ = d.client.Delete(key)
}

func (d *turboCacheMemcachedDriver) has(key string) bool {
	_, err := d.client.Get(key)
	return err == nil
}

func (d *turboCacheMemcachedDriver) flush() {
	_ = d.client.FlushAll()
}

// turboCacheFileDriver is a simple file-based cache (not for production, demo only).
type turboCacheFileDriver struct {
	root string
	mu   sync.Mutex
}

func newTurboCacheFileDriver(root string) *turboCacheFileDriver {
	if err := os.MkdirAll(root, 0750); err != nil {
		logger.Error("Failed to create cache directory: %v", err)
	}
	return &turboCacheFileDriver{root: root}
}

func (d *turboCacheFileDriver) get(key string) (any, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	// G304: path is always joined to controlled cache root, safe for inclusion
	// #nosec G304
	path := filepath.Join(d.root, key)
	// G304: path is always read from controlled cache root, safe for inclusion
	// #nosec G304
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	// Try to deserialize JSON, fallback to string if it fails
	var result any
	if err := json.Unmarshal(data, &result); err == nil {
		return result, true
	}
	return string(data), true
}

func (d *turboCacheFileDriver) set(key string, value any, _ int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	path := filepath.Join(d.root, key)

	// Serialize complex values to JSON
	var serializedValue []byte
	switch v := value.(type) {
	case string:
		serializedValue = []byte(v)
	case int, int64, float64, bool:
		serializedValue = []byte(fmt.Sprintf("%v", v))
	default:
		if jsonBytes, err := json.Marshal(v); err == nil {
			serializedValue = jsonBytes
		} else {
			serializedValue = []byte(fmt.Sprintf("%v", v))
		}
	}

	_ = os.WriteFile(path, serializedValue, 0600)
	// TTL not implemented for file driver
}

func (d *turboCacheFileDriver) del(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	path := filepath.Join(d.root, key)
	_ = os.Remove(path)
}

func (d *turboCacheFileDriver) has(key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	path := filepath.Join(d.root, key)
	_, err := os.Stat(path)
	return err == nil
}

func (d *turboCacheFileDriver) flush() {
	d.mu.Lock()
	defer d.mu.Unlock()
	files, _ := os.ReadDir(d.root)
	for _, f := range files {
		_ = os.Remove(filepath.Join(d.root, f.Name()))
	}
}

// TurboCacheUtils provides shared turboCache functionality and drivers.
type TurboCacheUtils struct {
	config  *config.CacheConfig
	drivers map[string]interface {
		get(key string) (any, bool)
		set(key string, value any, ttl int64)
		del(key string)
		has(key string) bool
		flush()
	}
}

// NewTurboCacheUtils creates a new turboCache utilities instance with configuration.
func NewTurboCacheUtils(cacheConfig *config.CacheConfig) *TurboCacheUtils {
	utils := &TurboCacheUtils{
		config: cacheConfig,
		drivers: make(map[string]interface {
			get(key string) (any, bool)
			set(key string, value any, ttl int64)
			del(key string)
			has(key string) bool
			flush()
		}),
	}

	// Initialize drivers from configuration
	for name, driverConfig := range cacheConfig.Drivers {
		switch driverConfig.Driver {
		case "memory":
			utils.drivers[name] = newTurboCacheMemoryDriver()
		case "redis":
			utils.drivers[name] = newTurboCacheRedisDriverWithConfig(driverConfig)
		case "memcached":
			utils.drivers[name] = newTurboCacheMemcachedDriver(driverConfig.Host, driverConfig.Port)
		case "file":
			root := driverConfig.Root
			if root == "" {
				root = "./cache"
			}
			utils.drivers[name] = newTurboCacheFileDriver(root)
		}
	}

	return utils
}

// getDriver returns the appropriate cache driver instance based on driver name.
func (tcu *TurboCacheUtils) getDriver(driverName string) interface {
	get(key string) (any, bool)
	set(key string, value any, ttl int64)
	del(key string)
	has(key string) bool
	flush()
} {
	if driverName == "" {
		driverName = tcu.config.Default
	}

	if driver, exists := tcu.drivers[driverName]; exists {
		return driver
	}

	// Fallback to default driver
	if defaultDriver, exists := tcu.drivers[tcu.config.Default]; exists {
		logger.Warn("Cache driver '%s' not found, using default '%s'", driverName, tcu.config.Default)
		return defaultDriver
	}

	// Ultimate fallback to memory driver if available
	for name, driver := range tcu.drivers {
		if driverConfig, exists := tcu.config.Drivers[name]; exists && driverConfig.Driver == "memory" {
			logger.Warn("Cache driver '%s' not found, using memory fallback '%s'", driverName, name)
			return driver
		}
	}

	logger.Error("No cache drivers available")
	return nil
}

// parseOptionsFromMap parses options from a map and returns key, value, ttl, driver, and operation.
func (tcu *TurboCacheUtils) parseOptionsFromMap(options map[string]any) (key string, value any, ttl int64, driver string, op string) {
	if k, ok := options["key"].(string); ok {
		key = k
	}
	if v, ok := options["value"]; ok {
		value = v
	}
	if t, ok := options["ttl"].(int64); ok {
		ttl = t
	} else if t, ok := options["ttl"].(float64); ok {
		ttl = int64(t)
	}
	if d, ok := options["driver"].(string); ok {
		driver = d
	}
	if o, ok := options["op"].(string); ok {
		op = o
	}
	return key, value, ttl, driver, op
}

// parsePositionalArgs parses positional arguments and returns key, value, ttl, driver, and operation.
func (tcu *TurboCacheUtils) parsePositionalArgs(args []goja.Value) (key string, value any, ttl int64, driver string, op string) {
	if len(args) > 0 {
		key = args[0].String()
	}

	// For set operations, we expect: key, value, ttl, options
	// For other operations, we expect: key, options
	argIndex := 1

	// Parse value (second argument - can be any type, including maps)
	if len(args) > argIndex {
		arg := args[argIndex]

		// If we have 4 arguments, the second one is definitely the value (set operation)
		// If we have 2 arguments and the second one is a map, it's options (get/del/has operation)
		if len(args) == 4 || (len(args) > 2 && arg.ExportType().Kind() != reflect.Map) {
			value = arg.Export()
			argIndex++
		}
	}

	// Parse ttl (third argument if present and numeric)
	if len(args) > argIndex {
		arg := args[argIndex]
		if arg.ExportType().Kind() != reflect.Map {
			if t, ok := arg.Export().(int64); ok {
				ttl = t
				argIndex++
			} else if t, ok := arg.Export().(float64); ok {
				ttl = int64(t)
				argIndex++
			}
		}
	}

	// Parse options map (should be the last argument)
	if len(args) > argIndex {
		arg := args[argIndex]
		if arg.ExportType().Kind() == reflect.Map {
			if opts, ok := arg.Export().(map[string]any); ok {
				if d, ok := opts["driver"].(string); ok {
					driver = d
				}
				if o, ok := opts["op"].(string); ok {
					op = o
				}
			}
		}
	}

	return key, value, ttl, driver, op
}

// ParseTurboCacheArgs parses turboCache arguments and returns key, value, ttl, driver, and operation.
func (tcu *TurboCacheUtils) ParseTurboCacheArgs(call goja.FunctionCall, _ *goja.Runtime) (key string, value any, ttl int64, driver string, op string, err error) {
	if len(call.Arguments) == 0 {
		err = fmt.Errorf("turboCache requires at least 1 argument (key or options object)")
		return "", nil, 0, "", "", err
	}

	firstArg := call.Argument(0)
	switch firstArg.ExportType().Kind() {
	case reflect.Map:
		options, ok := firstArg.Export().(map[string]any)
		if !ok {
			err = fmt.Errorf("options object must be a map")
			return "", nil, 0, "", "", err
		}
		key, value, ttl, driver, op = tcu.parseOptionsFromMap(options)

	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan,
		reflect.Func, reflect.Interface, reflect.Pointer, reflect.Slice, reflect.String, reflect.Struct, reflect.UnsafePointer:
		key, value, ttl, driver, op = tcu.parsePositionalArgs(call.Arguments)
	}

	if driver == "" {
		driver = tcu.config.Default
	}
	if op == "" {
		op = "get"
	}

	return key, value, ttl, driver, op, nil
}

// turboCacheAsync implements the async turboCache Goja binding (Promise-based)
// NOTE: The event loop manager must be passed in to use RunOnLoop for async Goja operations.
func (tcu *TurboCacheUtils) turboCacheAsync(call goja.FunctionCall, rt *goja.Runtime, eventLoop interface {
	RunOnLoop(func(*goja.Runtime)) bool
}) goja.Value {
	key, value, ttl, driver, op, err := tcu.ParseTurboCacheArgs(call, rt)
	if err != nil {
		panic(rt.NewGoError(err))
	}

	logger.Debug("turboCacheAsync called with key=%s, value=%v, ttl=%d, driver=%s, op=%s", key, value, ttl, driver, op)

	promise, resolve, reject := rt.NewPromise()

	go func() {
		var result any
		var opErr error

		cache := tcu.getDriver(driver)
		if cache == nil {
			opErr = fmt.Errorf("cache driver '%s' not available", driver)
		} else {
			switch op {
			case "get":
				result, _ = cache.get(key)
			case "set":
				cache.set(key, value, ttl)
				result = true
			case "del":
				cache.del(key)
				result = true
			case "has":
				result = cache.has(key)
			case "flush":
				cache.flush()
				result = true
			default:
				opErr = fmt.Errorf("unsupported turboCache op: %s", op)
			}
		}
		if opErr != nil {
			eventLoop.RunOnLoop(func(_ *goja.Runtime) {
				_ = reject(opErr)
			})
			return
		}
		eventLoop.RunOnLoop(func(_ *goja.Runtime) {
			_ = resolve(result)
		})
	}()

	return rt.ToValue(promise)
}
