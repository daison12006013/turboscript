package scheduler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
)

// TestScheduler_BasicFunctionality tests basic scheduler functionality without database
func TestScheduler_BasicFunctionality(t *testing.T) {
	// Create a temporary directory for test scheduler files
	tempDir, err := os.MkdirTemp("", "scheduler_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a simple test scheduler TypeScript file
	testHandlerContent := `
export async function handle(ctx) {
    const { payload, logger } = ctx;

    console.log('Test scheduler executed at: ' + new Date().toISOString());

    if (payload?.message) {
        console.log('Message: ' + payload.message);
    }

    return {
        success: true,
        message: 'Test scheduler completed successfully',
        executedAt: new Date().toISOString(),
        payload: payload
    };
}
`

	handlerPath := filepath.Join(tempDir, "test-handler.ts")
	err = os.WriteFile(handlerPath, []byte(testHandlerContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test handler: %v", err)
	}

	// Setup scheduler configuration
	schedulerConfig := config.SchedulerConfig{
		Enabled:  true,
		Timezone: "UTC",
		Path:     tempDir,
		LogLevel: "info",
		Tasks: []config.SchedulerTaskConfig{
			{
				Name:        "test-basic-task",
				Cron:        "0 0 2 * * *", // Daily at 2 AM (won't actually run during test)
				Handler:     "test-handler.ts",
				Enabled:     true,
				Timezone:    "UTC",
				Timeout:     10,
				Description: "Basic functionality test task",
				Payload: map[string]any{
					"message": "Hello from basic test",
					"testId":  "test-123",
				},
				Environment: map[string]string{
					"TEST_ENV": "basic",
				},
			},
		},
	}

	// Create and start scheduler without database and TypeScript config
	manager := NewManager(schedulerConfig, nil, nil)

	err = manager.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Verify the scheduler started
	if !manager.running {
		t.Error("Expected scheduler to be running")
	}

	// Verify the handler file exists
	if _, err := os.Stat(handlerPath); os.IsNotExist(err) {
		t.Error("Expected handler file to exist")
	}

	// Stop the scheduler
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop scheduler: %v", err)
	}

	// Verify the scheduler stopped
	if manager.running {
		t.Error("Expected scheduler to be stopped")
	}

	t.Log("Basic functionality test completed successfully")
}

// TestScheduler_ConfigurationValidation tests configuration validation
func TestScheduler_ConfigurationValidation(t *testing.T) {
	testCases := []struct {
		name        string
		config      config.SchedulerConfig
		expectError bool
	}{
		{
			name: "Valid Configuration",
			config: config.SchedulerConfig{
				Enabled:  true,
				Timezone: "UTC",
				Path:     "./test/schedulers",
				LogLevel: "info",
				Tasks: []config.SchedulerTaskConfig{
					{
						Name:        "valid-task",
						Cron:        "0 0 2 * * *",
						Handler:     "handler.ts",
						Enabled:     true,
						Timezone:    "UTC",
						Timeout:     300,
						Description: "Valid test task",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Empty Path",
			config: config.SchedulerConfig{
				Enabled:  true,
				Timezone: "UTC",
				Path:     "",
				LogLevel: "info",
				Tasks: []config.SchedulerTaskConfig{
					{
						Name:    "test-task",
						Cron:    "0 0 2 * * *",
						Handler: "handler.ts",
						Enabled: true,
					},
				},
			},
			expectError: false, // NewManager shouldn't fail on empty path
		},
		{
			name: "Invalid Timezone",
			config: config.SchedulerConfig{
				Enabled:  true,
				Timezone: "Invalid/Timezone",
				Path:     "./test/schedulers",
				LogLevel: "info",
				Tasks: []config.SchedulerTaskConfig{
					{
						Name:    "test-task",
						Cron:    "0 0 2 * * *",
						Handler: "handler.ts",
						Enabled: true,
					},
				},
			},
			expectError: false, // Scheduler gracefully handles invalid timezones by warning and continuing
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager(tc.config, nil, nil)

			err := manager.Start()

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Always try to stop, even if start failed
			_ = manager.Stop()
		})
	}
}
