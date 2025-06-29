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

// Package plugins provides a pluggable system for extending TurboScript with custom functions.
//
// The plugin system allows developers to register custom JavaScript functions that can be
// called from TypeScript route handlers. Plugins are automatically loaded and configured
// based on the turboscript.yml configuration file.
package plugins

import (
	"fmt"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// Plugin represents a TurboScript plugin that can be loaded into the JavaScript runtime.
type Plugin interface {
	// Name returns the unique name/identifier for this plugin
	Name() string

	// Initialize sets up the plugin with the provided configuration
	Initialize(config map[string]any) error

	// Register adds the plugin's functions to the JavaScript runtime
	Register(runtime *goja.Runtime, registry *require.Registry) error

	// Description returns a human-readable description of what this plugin does
	Description() string

	// Version returns the plugin version
	Version() string
}

// PluginConfig represents plugin configuration from turboscript.yml.
type PluginConfig struct {
	Name    string         `yaml:"name"`    // Plugin name/identifier
	Enabled bool           `yaml:"enabled"` // Whether the plugin is enabled
	Config  map[string]any `yaml:"config"`  // Plugin-specific configuration
}

// Manager handles plugin registration and lifecycle management.
type Manager struct {
	plugins     map[string]Plugin
	configs     map[string]PluginConfig
	mu          sync.RWMutex
	initialized bool
}

// NewManager creates a new plugin manager instance.
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		configs: make(map[string]PluginConfig),
	}
}

// RegisterPlugin registers a new plugin with the manager.
func (m *Manager) RegisterPlugin(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := plugin.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin '%s' is already registered", name)
	}

	m.plugins[name] = plugin
	return nil
}

// LoadConfigs loads plugin configurations from the provided config map.
func (m *Manager) LoadConfigs(pluginConfigs []PluginConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, config := range pluginConfigs {
		m.configs[config.Name] = config
	}

	return nil
}

// InitializePlugins initializes all enabled plugins with their configurations.
func (m *Manager) InitializePlugins() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, plugin := range m.plugins {
		config, exists := m.configs[name]
		if !exists {
			// Plugin not configured, skip
			continue
		}

		if !config.Enabled {
			// Plugin disabled, skip
			continue
		}

		if err := plugin.Initialize(config.Config); err != nil {
			return fmt.Errorf("failed to initialize plugin '%s': %w", name, err)
		}
	}

	m.initialized = true
	return nil
}

// RegisterWithRuntime registers all enabled plugins with the JavaScript runtime.
func (m *Manager) RegisterWithRuntime(runtime *goja.Runtime, registry *require.Registry) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.initialized {
		return fmt.Errorf("plugins must be initialized before registering with runtime")
	}

	for name, plugin := range m.plugins {
		config, exists := m.configs[name]
		if !exists || !config.Enabled {
			continue
		}

		if err := plugin.Register(runtime, registry); err != nil {
			return fmt.Errorf("failed to register plugin '%s' with runtime: %w", name, err)
		}
	}

	return nil
}

// GetPlugin returns a plugin by name.
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	return plugin, exists
}

// ListPlugins returns a list of all registered plugins with their status.
func (m *Manager) ListPlugins() map[string]PluginStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]PluginStatus)

	for name, plugin := range m.plugins {
		config, hasConfig := m.configs[name]
		enabled := hasConfig && config.Enabled

		status[name] = PluginStatus{
			Name:        name,
			Description: plugin.Description(),
			Version:     plugin.Version(),
			Enabled:     enabled,
			Configured:  hasConfig,
		}
	}

	return status
}

// PluginStatus represents the status of a plugin.
type PluginStatus struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Enabled     bool   `json:"enabled"`
	Configured  bool   `json:"configured"`
}

// GlobalManager is the default plugin manager instance.
var GlobalManager = NewManager()

// RegisterGlobalPlugin registers a plugin with the global manager.
func RegisterGlobalPlugin(plugin Plugin) error {
	return GlobalManager.RegisterPlugin(plugin)
}
