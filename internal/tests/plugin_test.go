package main

import (
	"strings"
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/plugins"
	"github.com/daison12006013/turboscript/internal/plugins/fileupload"
	"github.com/daison12006013/turboscript/internal/tsengine"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func TestPluginSystem(t *testing.T) {
	// Create a new runtime
	rt := goja.New()
	registry := require.NewRegistry()

	// Initialize built-in plugins (like main.go does)
	plugins.InitializeBuiltinPlugins()

	// Load plugin configuration (like main.go does)
	type PluginConfig struct {
		Name    string                 `yaml:"name"`
		Enabled bool                   `yaml:"enabled"`
		Options map[string]interface{} `yaml:"options"`
	}

	pluginConfigs := []PluginConfig{
		{
			Name:    "fileupload",
			Enabled: true,
			Options: map[string]interface{}{
				"upload_dir":    "./test_uploads",
				"max_file_size": 1048576, // 1MB
				"allowed_types": []string{"text/plain", "image/jpeg"},
			},
		},
	}

	// Convert to the right type for InitializePluginsWithConfig
	configPlugins := make([]config.PluginConfig, len(pluginConfigs))
	for i, cfg := range pluginConfigs {
		configPlugins[i] = config.PluginConfig{
			Name:    cfg.Name,
			Enabled: cfg.Enabled,
			Options: cfg.Options,
		}
	}

	// Initialize plugins with configuration
	err := plugins.InitializePluginsWithConfig(configPlugins)
	if err != nil {
		t.Fatalf("Failed to initialize plugins with config: %v", err)
	}

	// Check if plugin is actually registered and enabled
	pluginList := plugins.GlobalManager.ListPlugins()
	t.Logf("Available plugins: %+v", pluginList)

	if fileuploadPlugin, exists := pluginList["fileupload"]; !exists {
		t.Fatalf("fileupload plugin not found in plugin list")
	} else {
		t.Logf("fileupload plugin status: %+v", fileuploadPlugin)
	}

	// Enable require BEFORE registering plugins with runtime
	registry.Enable(rt)

	// Register with runtime
	err = plugins.GlobalManager.RegisterWithRuntime(rt, registry)
	if err != nil {
		t.Fatalf("Failed to register with runtime: %v", err)
	}

	// Test that the plugin is available via require
	result, err := rt.RunString(`
		let result = '';

		// First test global access
		try {
			if (typeof fileUploadGlobal !== 'undefined') {
				result += 'GLOBAL: fileUploadGlobal found; ';
				if (fileUploadGlobal.saveBase64) {
					result += 'Global saveBase64: ' + fileUploadGlobal.saveBase64() + '; ';
				}
			} else {
				result += 'GLOBAL: fileUploadGlobal not found; ';
			}
		} catch (e) {
			result += 'GLOBAL ERROR: ' + e.message + '; ';
		}

		// Then test require
		try {
			const fileUpload = require('fileupload');
			if (!fileUpload) {
				result += 'REQUIRE ERROR: fileupload plugin returned null/undefined; ';
			} else {
				// Check available functions
				const keys = Object.keys(fileUpload);

				if (keys.length === 0) {
					result += 'REQUIRE ERROR: fileupload module is empty object; ';
				} else if (!fileUpload.saveBase64) {
					result += 'REQUIRE ERROR: saveBase64 function not found. Available: ' + keys.join(', ') + '; ';
				} else {
					result += 'REQUIRE SUCCESS: Plugin system working! Available functions: ' + keys.join(', ') + '; ';
				}
			}
		} catch (e) {
			result += 'REQUIRE ERROR: Exception during require: ' + e.message + '; ';
		}

		result;
	`)

	if err != nil {
		t.Fatalf("Plugin test failed: %v", err)
	}

	t.Logf("Result: %s", result.String())

	// Only fail if neither global nor require worked
	if !strings.Contains(result.String(), "SUCCESS") && !strings.Contains(result.String(), "Global saveBase64") {
		t.Fatalf("Plugin test did not succeed: %s", result.String())
	}
}

func TestTurboPluginFunction(t *testing.T) {
	// Create a new runtime
	rt := goja.New()
	registry := require.NewRegistry()

	// Register the fileupload plugin only if it's not already registered
	uploadPlugin := fileupload.NewFileUploadPlugin()
	_, exists := plugins.GlobalManager.GetPlugin("fileupload")
	if !exists {
		err := plugins.GlobalManager.RegisterPlugin(uploadPlugin)
		if err != nil {
			t.Fatalf("Failed to register plugin: %v", err)
		}
	}

	// Load plugin configuration
	configs := []plugins.PluginConfig{
		{
			Name:    "fileupload",
			Enabled: true,
			Config: map[string]any{
				"upload_dir":    "./test_uploads",
				"max_file_size": 1048576, // 1MB
				"allowed_types": []string{"text/plain", "image/jpeg"},
			},
		},
	}

	err := plugins.GlobalManager.LoadConfigs(configs)
	if err != nil {
		t.Fatalf("Failed to load configs: %v", err)
	}

	// Initialize plugins
	err = plugins.GlobalManager.InitializePlugins()
	if err != nil {
		t.Fatalf("Failed to initialize plugins: %v", err)
	}

	// Register with runtime
	err = plugins.GlobalManager.RegisterWithRuntime(rt, registry)
	if err != nil {
		t.Fatalf("Failed to register with runtime: %v", err)
	}

	// Enable require
	registry.Enable(rt)

	// Register turboPlugin function
	tsengine.RegisterSharedPluginModule(rt, registry)

	// Test that turboPlugin function works
	result, err := rt.RunString(`
		try {
			const fileUpload = turboPlugin('fileupload');
			if (!fileUpload) {
				'ERROR: turboPlugin returned null';
			} else if (!fileUpload.saveBase64) {
				'ERROR: saveBase64 function not found via turboPlugin';
			} else {
				'SUCCESS: turboPlugin function working!';
			}
		} catch (e) {
			'ERROR: turboPlugin failed: ' + e.message;
		}
	`)

	if err != nil {
		t.Fatalf("turboPlugin test failed: %v", err)
	}

	t.Logf("turboPlugin Result: %s", result.String())

	if !strings.Contains(result.String(), "SUCCESS") {
		t.Fatalf("turboPlugin test did not succeed: %s", result.String())
	}
}
