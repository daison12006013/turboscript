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

// Package plugins provides initialization for built-in TurboScript plugins.
package plugins

import (
	"log"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/plugins/fileupload"
)

// InitializeBuiltinPlugins registers all built-in plugins with the global manager.
// This should be called during application startup.
func InitializeBuiltinPlugins() {
	// Register file upload plugin
	if err := RegisterGlobalPlugin(fileupload.NewFileUploadPlugin()); err != nil {
		log.Printf("Failed to register file upload plugin: %v", err)
	}

	// Future built-in plugins can be registered here
	// if err := RegisterGlobalPlugin(NewOtherPlugin()); err != nil {
	//     log.Printf("Failed to register other plugin: %v", err)
	// }
}

// InitializePluginsWithConfig initializes plugins with configuration and starts them.
func InitializePluginsWithConfig(pluginConfigs []config.PluginConfig) error {
	// Convert config.PluginConfig to plugins.PluginConfig
	configs := make([]PluginConfig, len(pluginConfigs))
	for i, cfg := range pluginConfigs {
		configs[i] = PluginConfig{
			Name:    cfg.Name,
			Enabled: cfg.Enabled,
			Config:  cfg.Options,
		}
	}

	// Load plugin configurations
	if err := GlobalManager.LoadConfigs(configs); err != nil {
		return err
	}

	// Initialize all plugins
	if err := GlobalManager.InitializePlugins(); err != nil {
		return err
	}

	return nil
}
