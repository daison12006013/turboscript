package scheduler

import (
	"testing"

	"github.com/daison12006013/turboscript/internal/config"
)

func TestNewManager(t *testing.T) {
	// Test creating a manager with basic configuration
	schedulerConfig := config.SchedulerConfig{
		Enabled:  true,
		Timezone: "UTC",
		Path:     "./test/schedulers",
		LogLevel: "info",
		Tasks: []config.SchedulerTaskConfig{
			{
				Name:        "test-task",
				Cron:        "0 0 2 * * *",
				Handler:     "test-handler.ts",
				Enabled:     true,
				Timezone:    "UTC",
				Timeout:     300,
				Description: "Test scheduler task",
			},
		},
	}

	manager := NewManager(schedulerConfig, nil, nil)

	if manager == nil {
		t.Fatal("Expected NewManager to return a non-nil manager")
	}

	if manager.config.Enabled != true {
		t.Error("Expected manager config enabled to be true")
	}

	if manager.config.Timezone != "UTC" {
		t.Error("Expected manager config timezone to be UTC")
	}

	if len(manager.config.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(manager.config.Tasks))
	}

	if manager.running {
		t.Error("Expected manager to not be running initially")
	}
}

func TestManager_StartStop(t *testing.T) {
	// Test starting and stopping the manager
	schedulerConfig := config.SchedulerConfig{
		Enabled:  true,
		Timezone: "UTC",
		Path:     "./test/schedulers",
		LogLevel: "info",
		Tasks: []config.SchedulerTaskConfig{
			{
				Name:        "test-start-stop-task",
				Cron:        "0 0 2 * * *",
				Handler:     "test-handler.ts",
				Enabled:     true,
				Timezone:    "UTC",
				Timeout:     300,
				Description: "Test start/stop task",
			},
		},
	}

	manager := NewManager(schedulerConfig, nil, nil)

	// Test Start
	err := manager.Start()
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	if !manager.running {
		t.Error("Expected manager to be running after Start()")
	}

	// Test stopping
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop manager: %v", err)
	}

	if manager.running {
		t.Error("Expected manager to not be running after Stop()")
	}
}

func TestManager_StartWhenDisabled(t *testing.T) {
	// Test that starting a disabled scheduler returns without error
	schedulerConfig := config.SchedulerConfig{
		Enabled:  false,
		Timezone: "UTC",
		Path:     "./test/schedulers",
		LogLevel: "info",
		Tasks:    []config.SchedulerTaskConfig{},
	}

	manager := NewManager(schedulerConfig, nil, nil)

	err := manager.Start()
	if err != nil {
		t.Fatalf("Expected no error when starting disabled scheduler, got: %v", err)
	}

	if manager.running {
		t.Error("Expected manager to not be running when disabled")
	}
}

func TestManager_MultipleStartStop(t *testing.T) {
	// Test multiple start/stop cycles
	schedulerConfig := config.SchedulerConfig{
		Enabled:  true,
		Timezone: "UTC",
		Path:     "./test/schedulers",
		LogLevel: "info",
		Tasks: []config.SchedulerTaskConfig{
			{
				Name:        "test-multiple-task",
				Cron:        "0 0 2 * * *",
				Handler:     "test-handler.ts",
				Enabled:     true,
				Timezone:    "UTC",
				Timeout:     300,
				Description: "Test multiple start/stop task",
			},
		},
	}

	manager := NewManager(schedulerConfig, nil, nil)

	// First cycle
	err := manager.Start()
	if err != nil {
		t.Fatalf("Failed to start manager first time: %v", err)
	}

	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop manager first time: %v", err)
	}

	// Second cycle
	err = manager.Start()
	if err != nil {
		t.Fatalf("Failed to start manager second time: %v", err)
	}

	err = manager.Stop()
	if err != nil {
		t.Fatalf("Failed to stop manager second time: %v", err)
	}

	if manager.running {
		t.Error("Expected manager to not be running after multiple cycles")
	}
}
