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

// Package scheduler provides cron-like scheduled task functionality for TurboScript.
//
// The scheduler allows developers to define recurring tasks using standard cron expressions
// that execute TypeScript handlers at specified intervals. This is useful for:
// - Data retention cleanup
// - Report generation
// - Cache warming
// - Periodic data processing
// - Maintenance tasks
//
// Tasks are configured in turboscript.yml and handlers are TypeScript files that export
// a handle(event: Event) function, similar to background jobs but triggered by schedule
// rather than manual dispatch.
package scheduler

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/tsengine"
)

// Manager manages scheduled tasks using cron expressions.
type Manager struct {
	config   config.SchedulerConfig
	cron     *cron.Cron
	db       *sql.DB
	tsConfig *config.TypeScriptCompilerConfig
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	running  bool
	tasks    map[string]*ScheduledTask
}

// ScheduledTask represents a single scheduled task with its execution context.
type ScheduledTask struct {
	ID         cron.EntryID
	Config     config.SchedulerTaskConfig
	NextRun    time.Time
	LastRun    *time.Time
	RunCount   int64
	ErrorCount int64
	LastError  error
	IsRunning  bool
	mu         sync.RWMutex
}

// TaskExecution represents a single execution of a scheduled task.
type TaskExecution struct {
	TaskName  string
	StartTime time.Time
	EndTime   *time.Time
	Duration  time.Duration
	Success   bool
	Error     error
	Output    any
}

// NewManager creates a new scheduler manager.
func NewManager(cfg config.SchedulerConfig, db *sql.DB, tsConfig *config.TypeScriptCompilerConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	// Create cron instance with proper timezone support
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		logger.Warn("Invalid timezone '%s', using UTC: %v", cfg.Timezone, err)
		loc = time.UTC
	}

	cronInstance := cron.New(
		cron.WithLocation(loc),
		cron.WithSeconds(), // Support second-level precision
	)

	return &Manager{
		config:   cfg,
		cron:     cronInstance,
		db:       db,
		tsConfig: tsConfig,
		ctx:      ctx,
		cancel:   cancel,
		tasks:    make(map[string]*ScheduledTask),
	}
}

// Start starts the scheduler and registers all configured tasks.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("scheduler is already running")
	}

	if !m.config.Enabled {
		logger.Info("Scheduler is disabled in configuration")
		return nil
	}

	logger.Info("Starting scheduler with timezone: %s", m.config.Timezone)

	// Register all configured tasks
	for _, taskConfig := range m.config.Tasks {
		if err := m.registerTask(taskConfig); err != nil {
			logger.Error("Failed to register task '%s': %v", taskConfig.Name, err)
			continue
		}
	}

	// Start the cron scheduler
	m.cron.Start()
	m.running = true

	logger.Info("Scheduler started with %d tasks", len(m.tasks))
	m.logNextExecutions()

	return nil
}

// Stop stops the scheduler and cancels all running tasks.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	logger.Info("Stopping scheduler...")

	// Stop the cron scheduler
	cronCtx := m.cron.Stop()

	// Cancel our context to stop any running tasks
	m.cancel()

	// Wait for cron to finish current executions
	<-cronCtx.Done()

	m.running = false
	logger.Info("Scheduler stopped")

	return nil
}

// registerTask registers a single task with the cron scheduler.
func (m *Manager) registerTask(taskConfig config.SchedulerTaskConfig) error {
	if !taskConfig.Enabled {
		logger.Debug("Task '%s' is disabled, skipping registration", taskConfig.Name)
		return nil
	}

	if taskConfig.Cron == "" {
		return fmt.Errorf("task '%s' has empty cron expression", taskConfig.Name)
	}

	if taskConfig.Handler == "" {
		return fmt.Errorf("task '%s' has empty handler path", taskConfig.Name)
	}

	// Parse timezone for this specific task
	loc, err := time.LoadLocation(taskConfig.Timezone)
	if err != nil {
		logger.Warn("Invalid timezone '%s' for task '%s', using scheduler default: %v",
			taskConfig.Timezone, taskConfig.Name, err)
		loc, _ = time.LoadLocation(m.config.Timezone)
	}

	// Create the scheduled task
	task := &ScheduledTask{
		Config: taskConfig,
	}

	// Create a wrapper function that handles the task execution
	taskFunc := func() {
		m.executeTask(task)
	}

	// Add the task to cron with its specific timezone
	entryID, err := m.cron.AddFunc(taskConfig.Cron, taskFunc)
	if err != nil {
		return fmt.Errorf("failed to add cron job for task '%s': %w", taskConfig.Name, err)
	}

	task.ID = entryID

	// Calculate next run time
	schedule, err := cron.ParseStandard(taskConfig.Cron)
	if err == nil {
		task.NextRun = schedule.Next(time.Now().In(loc))
	}

	m.tasks[taskConfig.Name] = task

	logger.Info("Registered task '%s' with cron '%s' (timezone: %s, next run: %s)",
		taskConfig.Name, taskConfig.Cron, taskConfig.Timezone, task.NextRun.Format(time.RFC3339))

	return nil
}

// executeTask executes a single scheduled task.
func (m *Manager) executeTask(task *ScheduledTask) {
	task.mu.Lock()
	if task.IsRunning {
		logger.Warn("Task '%s' is already running, skipping execution", task.Config.Name)
		task.mu.Unlock()
		return
	}
	task.IsRunning = true
	task.mu.Unlock()

	defer func() {
		task.mu.Lock()
		task.IsRunning = false
		task.mu.Unlock()
	}()

	execution := &TaskExecution{
		TaskName:  task.Config.Name,
		StartTime: time.Now(),
	}

	logger.Info("Executing scheduled task: %s", task.Config.Name)

	// Execute the task with timeout
	ctx, cancel := context.WithTimeout(m.ctx, time.Duration(task.Config.Timeout)*time.Second)
	defer cancel()

	// Execute the TypeScript handler
	err := m.executeTaskHandler(ctx, task, execution)

	execution.EndTime = &[]time.Time{time.Now()}[0]
	execution.Duration = execution.EndTime.Sub(execution.StartTime)
	execution.Success = err == nil
	execution.Error = err

	// Update task statistics
	task.mu.Lock()
	task.RunCount++
	task.LastRun = &execution.StartTime
	if err != nil {
		task.ErrorCount++
		task.LastError = err
	} else {
		task.LastError = nil
	}

	// Update next run time
	schedule, parseErr := cron.ParseStandard(task.Config.Cron)
	if parseErr == nil {
		loc, _ := time.LoadLocation(task.Config.Timezone)
		task.NextRun = schedule.Next(time.Now().In(loc))
	}
	task.mu.Unlock()

	// Log execution result
	if err != nil {
		logger.Error("Task '%s' failed after %v: %v", task.Config.Name, execution.Duration, err)
	} else {
		logger.Info("Task '%s' completed successfully in %v", task.Config.Name, execution.Duration)
	}
}

// executeTaskHandler executes the TypeScript handler for a scheduled task.
func (m *Manager) executeTaskHandler(ctx context.Context, task *ScheduledTask, execution *TaskExecution) error {
	// Build the full path to the handler
	handlerPath := filepath.Join(m.config.Path, task.Config.Handler)
	if !filepath.IsAbs(handlerPath) {
		handlerPath = filepath.Join(".", handlerPath)
	}

	// Create an isolated TypeScript executor for this task execution
	fileResolver := tsengine.GetResolverFromConfig(false, false)                                             // Use default resolver
	isolatedExecutor := tsengine.NewIsolatedTSExecutorWithResolverAndConfig(false, fileResolver, m.tsConfig) // Don't preserve response for schedulers

	// Ensure cleanup of the executor
	defer func() {
		if isolatedExecutor != nil {
			isolatedExecutor.TerminateAsync()
		}
	}()

	// Create the event object similar to background jobs
	eventData := map[string]any{
		"task_name":      task.Config.Name,
		"cron":           task.Config.Cron,
		"timezone":       task.Config.Timezone,
		"run_count":      task.RunCount + 1, // +1 because we haven't incremented it yet
		"scheduled_time": execution.StartTime.Format(time.RFC3339),
		"payload":        task.Config.Payload,
	}

	// Merge task-specific environment variables
	env := make(map[string]string)
	if task.Config.Environment != nil {
		for k, v := range task.Config.Environment {
			env[k] = v
		}
	}

	// Execute the TypeScript handler with timeout
	timeoutSeconds := task.Config.Timeout
	response, err := isolatedExecutor.ExecuteHandleAsyncWithTimeout(handlerPath, eventData, timeoutSeconds)
	if err != nil {
		return fmt.Errorf("failed to execute handler: %w", err)
	}

	execution.Output = response
	return nil
}

// GetTaskStatus returns the current status of all registered tasks.
func (m *Manager) GetTaskStatus() map[string]*ScheduledTask {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]*ScheduledTask)
	for name, task := range m.tasks {
		task.mu.RLock()
		// Create a copy to avoid race conditions
		taskCopy := &ScheduledTask{
			ID:         task.ID,
			Config:     task.Config,
			NextRun:    task.NextRun,
			RunCount:   task.RunCount,
			ErrorCount: task.ErrorCount,
			IsRunning:  task.IsRunning,
		}
		if task.LastRun != nil {
			lastRun := *task.LastRun
			taskCopy.LastRun = &lastRun
		}
		if task.LastError != nil {
			taskCopy.LastError = fmt.Errorf("%s", task.LastError.Error())
		}
		task.mu.RUnlock()

		status[name] = taskCopy
	}

	return status
}

// IsRunning returns true if the scheduler is currently running.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// logNextExecutions logs the next execution time for all registered tasks.
func (m *Manager) logNextExecutions() {
	if len(m.tasks) == 0 {
		return
	}

	logger.Info("Scheduled task execution times:")
	for name, task := range m.tasks {
		task.mu.RLock()
		logger.Info("  %s: %s (%s)", name, task.NextRun.Format(time.RFC3339), task.Config.Cron)
		task.mu.RUnlock()
	}
}

// TriggerTask manually triggers a specific task for testing or administrative purposes.
func (m *Manager) TriggerTask(taskName string) error {
	m.mu.RLock()
	task, exists := m.tasks[taskName]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task '%s' not found", taskName)
	}

	if !m.running {
		return fmt.Errorf("scheduler is not running")
	}

	// Execute the task in a goroutine to avoid blocking
	go m.executeTask(task)

	logger.Info("Manually triggered task: %s", taskName)
	return nil
}
