package main

import (
	"strings"
	"testing"

	"github.com/daison12006013/turboscript/internal/plugins"
	"github.com/daison12006013/turboscript/internal/plugins/fileupload"
	"github.com/daison12006013/turboscript/internal/tsengine"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

func TestTurboPluginFixValidation(t *testing.T) {
	// This test validates that the turboPlugin function works correctly
	// after the fix to register plugins in the async runtime path.

	// Clean up any existing plugins first
	plugins.GlobalManager = plugins.NewManager()

	// Register the fileupload plugin
	uploadPlugin := fileupload.NewFileUploadPlugin()
	err := plugins.GlobalManager.RegisterPlugin(uploadPlugin)
	if err != nil {
		t.Fatalf("Failed to register plugin: %v", err)
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

	err = plugins.GlobalManager.LoadConfigs(configs)
	if err != nil {
		t.Fatalf("Failed to load configs: %v", err)
	}

	// Initialize plugins
	err = plugins.GlobalManager.InitializePlugins()
	if err != nil {
		t.Fatalf("Failed to initialize plugins: %v", err)
	}

	// Test async runtime setup (mimics what happens in production)
	rt := goja.New()
	registry := require.NewRegistry()

	// Register plugins with runtime (this is the fix)
	err = plugins.GlobalManager.RegisterWithRuntime(rt, registry)
	if err != nil {
		t.Fatalf("Failed to register plugins with runtime: %v", err)
	}
	registry.Enable(rt)                               // Enable require() function for plugins
	tsengine.RegisterSharedPluginModule(rt, registry) // Register turboPlugin function

	// Test that turboPlugin function works correctly
	result, err := rt.RunString(`
		try {
			const fileUpload = turboPlugin('fileupload');
			if (!fileUpload) {
				'ERROR: turboPlugin returned null';
			} else if (!fileUpload.saveBase64) {
				'ERROR: saveBase64 function not found via turboPlugin';
			} else {
				'SUCCESS: turboPlugin function working correctly!';
			}
		} catch (e) {
			'ERROR: turboPlugin failed: ' + e.message;
		}
	`)

	if err != nil {
		t.Fatalf("turboPlugin test failed: %v", err)
	}

	t.Logf("Test Result: %s", result.String())

	if !strings.Contains(result.String(), "SUCCESS") {
		t.Fatalf("turboPlugin test did not succeed: %s", result.String())
	}

	t.Log("✅ turboPlugin fix validation passed!")
}
