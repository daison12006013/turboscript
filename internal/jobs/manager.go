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

// Package jobs provides background job processing capabilities for TurboScript.
//
// This package implements a job queue system that allows TypeScript handlers
// to dispatch background jobs that are processed asynchronously by Go workers.
// Jobs are executed in separate goroutines and can retry on failure.
package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/daison12006013/turboscript/internal/config"
	"github.com/daison12006013/turboscript/internal/email"
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/tsengine"
)

// Job represents a background job to be processed.
type Job struct {
	UID            string         `json:"uid"`             // UUID primary key
	Type           string         `json:"type"`            // Job type/handler name
	JobPath        string         `json:"job_path"`        // Path to the TypeScript job file
	Payload        map[string]any `json:"payload"`         // Job payload data
	Status         string         `json:"status"`          // pending, processing, completed, failed, cancelled
	Priority       int            `json:"priority"`        // Job priority (higher = more priority)
	RetryCount     int            `json:"retry_count"`     // Current retry attempt count
	MaxRetries     int            `json:"max_retries"`     // Maximum retry attempts allowed
	NextRetryAt    *time.Time     `json:"next_retry_at"`   // When to retry next
	ScheduledAt    time.Time      `json:"scheduled_at"`    // When job was scheduled
	StartedAt      *time.Time     `json:"started_at"`      // When job processing started
	CompletedAt    *time.Time     `json:"completed_at"`    // When job was completed/failed
	ErrorMessage   *string        `json:"error_message"`   // Error message if job failed
	ErrorDetails   map[string]any `json:"error_details"`   // Detailed error information
	TimeoutSeconds int            `json:"timeout_seconds"` // Job timeout in seconds
	WorkerID       *string        `json:"worker_id"`       // ID of worker processing the job
	CreatedAt      time.Time      `json:"created_at"`      // Job creation time
	UpdatedAt      time.Time      `json:"updated_at"`      // Last update time
}

// JobHistoryEntry represents a job history record.
type JobHistoryEntry struct {
	UID           string         `json:"uid"`
	JobUID        string         `json:"job_uid"`
	AttemptNumber int            `json:"attempt_number"`
	Status        string         `json:"status"`
	WorkerID      *string        `json:"worker_id"`
	StartedAt     time.Time      `json:"started_at"`
	CompletedAt   *time.Time     `json:"completed_at"`
	DurationMs    *int           `json:"duration_ms"`
	ErrorMessage  *string        `json:"error_message"`
	ErrorDetails  map[string]any `json:"error_details"`
	Output        map[string]any `json:"output"`
	CreatedAt     time.Time      `json:"created_at"`
}

// JobResult represents the result of job processing.
type JobResult struct {
	JobUID       string         `json:"job_uid"`
	Success      bool           `json:"success"`
	Error        string         `json:"error,omitempty"`
	ErrorDetails map[string]any `json:"error_details,omitempty"`
	Output       map[string]any `json:"output,omitempty"`
	EndTime      time.Time      `json:"end_time"`
	DurationMs   int            `json:"duration_ms"`
}

// Manager manages the job processing system.
type Manager struct {
	config       *config.JobsConfig
	cacheConfig  *config.CacheConfig
	serverConfig *config.ServerConfig
	emailService *email.Service
	db           *sql.DB
	workers      []*Worker
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	mu           sync.RWMutex
	running      bool
	pollInterval time.Duration
}

// Worker represents a job worker goroutine.
type Worker struct {
	id               int
	manager          *Manager
	ctx              context.Context
	cancel           context.CancelFunc
	workerID         string
	isolatedTSEngine *tsengine.TSExecutor // Isolated TypeScript executor for this worker
}

// NewManagerWithEmail creates a new job manager instance with email service support.
func NewManagerWithEmail(cfg *config.JobsConfig, cacheConfig *config.CacheConfig, serverCfg *config.ServerConfig, db *sql.DB, emailService *email.Service) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:       cfg,
		cacheConfig:  cacheConfig,
		serverConfig: serverCfg,
		emailService: emailService,
		db:           db,
		ctx:          ctx,
		cancel:       cancel,
		pollInterval: 3 * time.Second, // Poll interval for checking new jobs
	}
}

// Start starts the job processing system.
func (m *Manager) Start() error {
	if !m.config.Enabled {
		logger.Info("Job processing is disabled")
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("job manager is already running")
	}

	logger.Info("Starting job manager with %d workers", m.config.MaxWorkers)

	// Start workers
	m.workers = make([]*Worker, m.config.MaxWorkers)
	for i := 0; i < m.config.MaxWorkers; i++ {
		workerID := fmt.Sprintf("worker-%d-%d", i+1, time.Now().UnixNano())

		// Create isolated TypeScript executor for this worker
		// This prevents contamination between job execution and HTTP request processing
		isolatedExecutor := tsengine.NewIsolatedTSExecutor(false) // Don't preserve response for jobs
		isolatedExecutor.SetDatabase(m.db)

		// Set cache configuration for turboCache operations
		if m.cacheConfig != nil {
			isolatedExecutor.SetCacheConfig(m.cacheConfig)
		}

		// Set up email service for the isolated executor if available
		if m.emailService != nil {
			isolatedExecutor.SetEmailService(m.emailService)
		}

		worker := &Worker{
			id:               i + 1,
			manager:          m,
			workerID:         workerID,
			isolatedTSEngine: isolatedExecutor,
		}
		worker.ctx, worker.cancel = context.WithCancel(m.ctx)
		m.workers[i] = worker

		m.wg.Add(1)
		go worker.start()
	}

	// Recover failed jobs that were being processed before server restart
	go m.recoverFailedJobs()

	m.running = true
	logger.Info("Job manager started successfully")
	return nil
}

// Stop stops the job processing system.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	logger.Info("Stopping job manager...")

	// Cancel context to signal all workers to stop
	m.cancel()

	// Cancel individual worker contexts
	for _, worker := range m.workers {
		worker.cancel()
	}

	// Wait for all workers to finish
	m.wg.Wait()

	m.running = false
	logger.Info("Job manager stopped successfully")
	return nil
}

// DispatchJob adds a job to the database for processing.
func (m *Manager) DispatchJob(jobPath string, payload map[string]any) error {
	_, err := m.DispatchJobWithReturn(jobPath, payload)
	return err
}

// DispatchJobWithReturn adds a job to the database for processing and returns the job UID.
func (m *Manager) DispatchJobWithReturn(jobPath string, payload map[string]any) (string, error) {
	if !m.config.Enabled {
		return "", fmt.Errorf("job processing is disabled")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.running {
		return "", fmt.Errorf("job manager is not running")
	}

	// Convert jobPath to type for better identification
	jobType := filepath.Base(jobPath)
	if ext := filepath.Ext(jobType); ext != "" {
		jobType = jobType[:len(jobType)-len(ext)]
	}

	// Build full path including the configured jobs path
	backgroundJobsPath := m.config.PathBackgroundJobs
	if backgroundJobsPath == "" {
		backgroundJobsPath = "app/queue"
	}
	fullJobPath := filepath.Join(backgroundJobsPath, jobPath)
	if !filepath.IsAbs(fullJobPath) {
		fullJobPath = filepath.Join(".", fullJobPath)
	}

	// Convert payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal job payload: %w", err)
	}

	// Insert job into database
	var jobUID string
	query := `
		INSERT INTO jobs
			(type, path, payload, status, priority, max_retries, scheduled_at, timeout_seconds)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING uid
	`
	err = m.db.QueryRow(
		query,
		jobType,
		fullJobPath,
		payloadJSON,
		"pending",
		0, // Default priority
		m.config.RetryAttempts,
		time.Now(),
		m.config.Timeout,
	).Scan(&jobUID)

	if err != nil {
		return "", fmt.Errorf("failed to insert job into database: %w", err)
	}

	logger.Debug("Job %s dispatched successfully", jobUID)
	return jobUID, nil
}

// ErrNoJobAvailable is returned when no jobs are available for processing.
var ErrNoJobAvailable = fmt.Errorf("no job available")

// start starts the worker processing loop.
func (w *Worker) start() {
	defer w.manager.wg.Done()

	// Ensure isolated TypeScript executor is properly cleaned up when worker stops
	defer func() {
		if w.isolatedTSEngine != nil {
			w.isolatedTSEngine.TerminateAsync()
		}
	}()

	logger.Debug("Worker %d (ID: %s) started", w.id, w.workerID)

	// Main worker loop
	for {
		select {
		case <-w.ctx.Done():
			logger.Debug("Worker %d stopped", w.id)
			return
		case <-time.After(w.manager.pollInterval): // Try to claim a job
			job, err := w.claimNextJob()
			if err != nil {
				// Only log errors that aren't "no job available"
				if !errors.Is(err, ErrNoJobAvailable) {
					logger.Error("Worker %d failed to claim job: %v", w.id, err)
				}
				continue
			}

			// Process the claimed job
			w.processJob(job)
		}
	}
}

// claimNextJob attempts to claim the next available job from the database.
func (w *Worker) claimNextJob() (*Job, error) {
	// Use a transaction for atomicity when claiming a job
	tx, err := w.manager.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var committed bool
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				logger.Debug("Failed to rollback transaction: %v", rollbackErr)
			}
		}
	}()

	// Get the next available job with proper locking
	query := `
		SELECT
			uid, type, path, payload, status, priority,
			retry_count, max_retries, scheduled_at, timeout_seconds
		FROM jobs
		WHERE status = 'pending'
			AND (next_retry_at IS NULL OR next_retry_at <= CURRENT_TIMESTAMP)
			AND scheduled_at <= CURRENT_TIMESTAMP
		ORDER BY priority DESC, scheduled_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	var job Job
	var payloadJSON []byte

	err = tx.QueryRow(query).Scan(
		&job.UID, &job.Type, &job.JobPath, &payloadJSON,
		&job.Status, &job.Priority, &job.RetryCount, &job.MaxRetries,
		&job.ScheduledAt, &job.TimeoutSeconds,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoJobAvailable
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query job: %w", err)
	}

	// Parse payload JSON
	if err := json.Unmarshal(payloadJSON, &job.Payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal job payload: %w", err)
	}

	// Mark the job as processing
	now := time.Now()
	job.Status = "processing"
	job.StartedAt = &now
	workerID := w.workerID
	job.WorkerID = &workerID

	// Update the job in the database
	_, err = tx.Exec(`
		UPDATE jobs
		SET
			status = $1,
			started_at = $2,
			worker_id = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE uid = $4
	`, job.Status, job.StartedAt, job.WorkerID, job.UID)

	if err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	// Create a job history entry for this attempt
	_, err = tx.Exec(`
		INSERT INTO job_history (
			job_uid, attempt_number, status,
			worker_id, started_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, CURRENT_TIMESTAMP
		)
	`,
		job.UID, job.RetryCount+1, "started",
		job.WorkerID, job.StartedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create job history entry: %w", err)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	logger.Debug("Worker %d claimed job %s", w.id, job.UID)
	return &job, nil
}

// processJob processes a single job.
func (w *Worker) processJob(job *Job) {
	logger.Debug("Worker %d processing job %s (attempt %d/%d)",
		w.id, job.UID, job.RetryCount+1, job.MaxRetries)

	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(job.TimeoutSeconds)*time.Second,
	)
	defer cancel()

	// Execute the job
	result := w.executeJob(ctx, job)
	result.JobUID = job.UID

	// Calculate duration
	durationMs := int(time.Since(startTime).Milliseconds())
	result.DurationMs = durationMs

	if result.Success {
		w.handleSuccessfulJob(job, result, durationMs)
	} else {
		w.handleFailedJob(job, result, durationMs)
	}
}

// executeJob executes a job using the TypeScript engine.
func (w *Worker) executeJob(_ context.Context, job *Job) *JobResult {
	result := &JobResult{
		JobUID:  job.UID,
		Success: false,
		EndTime: time.Now(),
		Output:  make(map[string]any),
	}

	// Resolve the TypeScript file path
	tsFilePath := w.resolveTSFilePath(job.JobPath)

	// Create event for job execution
	event := w.createJobEvent(job)

	// Execute the TypeScript job handler using isolated executor
	timeoutSeconds := job.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = w.manager.config.Timeout
	}

	response, err := w.isolatedTSEngine.ExecuteHandleAsyncWithTimeout(tsFilePath, event, timeoutSeconds)
	if err != nil {
		return w.createErrorResult(result, fmt.Sprintf("Job execution failed: %v", err), "execution_error", err.Error())
	}

	// Process the response
	return w.processJobResponse(result, response)
}

// resolveTSFilePath resolves the TypeScript file path for a job using the TSExecutor's file resolver.
func (w *Worker) resolveTSFilePath(jobPath string) string {
	// The jobPath now contains the full path including the configured background jobs path
	// We can use it directly, but ensure it's properly resolved

	// If the path is not absolute, make it relative to current directory
	if !filepath.IsAbs(jobPath) {
		jobPath = filepath.Join(".", jobPath)
	}

	// Use the isolated TSExecutor's file resolver to find the actual file (handles .ts/.js resolution)
	if w.isolatedTSEngine != nil {
		if resolvedPath, err := w.isolatedTSEngine.ResolveFileQuiet(jobPath); err == nil {
			return resolvedPath
		}
	}

	// Fallback to original path if resolution fails
	return jobPath
}

// createJobEvent creates a mock event for job execution.
func (w *Worker) createJobEvent(job *Job) map[string]any {
	return map[string]any{
		"headers":         map[string]string{},
		"queryParameters": map[string]string{},
		"pathParameters":  map[string]string{},
		"body":            job.Payload,
		"jobUID":          job.UID,
		"env":             map[string]string{}, // Environment variables could be added here
	}
}

// createErrorResult creates an error result for a job.
func (w *Worker) createErrorResult(result *JobResult, errorMsg, errorType, details string) *JobResult {
	result.Error = errorMsg
	result.ErrorDetails = map[string]any{
		"error": details,
		"type":  errorType,
	}
	return result
}

// processJobResponse processes the TypeScript execution response.
func (w *Worker) processJobResponse(result *JobResult, response []byte) *JobResult {
	if len(response) == 0 {
		result.Success = true
		return result
	}

	var responseMap map[string]any
	if err := json.Unmarshal(response, &responseMap); err != nil {
		result.Output["rawResponse"] = string(response)
		result.Success = true
		return result
	}

	// Check for error responses
	if w.isErrorResponse(responseMap) {
		return w.handleErrorResponse(result, responseMap)
	}

	// Extract successful response data
	w.extractResponseData(result, responseMap)
	result.Success = true
	return result
}

// isErrorResponse checks if the response indicates an error.
func (w *Worker) isErrorResponse(responseMap map[string]any) bool {
	if code, exists := responseMap["code"]; exists {
		if codeInt, ok := code.(float64); ok && codeInt >= 400 {
			return true
		}
	}
	return false
}

// handleErrorResponse processes an error response from TypeScript execution.
func (w *Worker) handleErrorResponse(result *JobResult, responseMap map[string]any) *JobResult {
	result.Error = "Job execution returned error response"
	result.ErrorDetails = map[string]any{
		"code": responseMap["code"],
	}

	responseData, exists := responseMap["response"]
	if !exists {
		return result
	}

	respMap, ok := responseData.(map[string]any)
	if !ok {
		return result
	}

	if message, exists := respMap["message"]; exists {
		result.Error = fmt.Sprintf("Job returned error: %v", message)
		result.ErrorDetails["message"] = message
	}

	if errorDetails, exists := respMap["error"]; exists {
		result.ErrorDetails["details"] = errorDetails
	}

	return result
}

// extractResponseData extracts data from a successful response.
func (w *Worker) extractResponseData(result *JobResult, responseMap map[string]any) {
	responseData, exists := responseMap["response"]
	if !exists {
		result.Output["raw"] = responseMap
		return
	}

	respMap, ok := responseData.(map[string]any)
	if !ok {
		result.Output["response"] = responseData
		return
	}

	if data, exists := respMap["data"]; exists {
		result.Output["data"] = data
	}
	result.Output["response"] = respMap
}

// Manager methods...

// recoverFailedJobs identifies and resets jobs that were processing when the server was stopped.
func (m *Manager) recoverFailedJobs() {
	logger.Info("Recovering failed jobs from previous shutdown...")

	// Find jobs marked as processing but with no active worker
	query := `
		UPDATE jobs
		SET status = 'pending',
			retry_count = retry_count + 1,
			next_retry_at = NULL,
			worker_id = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE status = 'processing'
			AND (retry_count < max_retries OR max_retries = -1)
		RETURNING uid, type, retry_count
	`

	rows, err := m.db.Query(query)
	if err != nil {
		logger.Error("Failed to recover jobs: %v", err)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error("Failed to close rows: %v", err)
		}
	}()

	count := 0
	for rows.Next() {
		var id int64
		var uid, jobType string
		var retryCount int
		err := rows.Scan(&id, &uid, &jobType, &retryCount)
		if err != nil {
			logger.Error("Failed to scan recovered job: %v", err)
			continue
		}

		// Record failure in job history
		errorMsg := "Server shutdown while job was processing"
		_, err = m.db.Exec(`
			INSERT INTO job_history (
				job_id, job_uid, attempt_number, status,
				started_at, error_message, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP
			)
		`,
			id, uid, retryCount, "failed",
			time.Now(), errorMsg,
		)

		if err != nil {
			logger.Error("Failed to record job history for recovered job %s: %v", uid, err)
		}

		count++
	}

	logger.Info("Recovered %d failed jobs", count)

	// Permanently fail jobs that exceeded retry count
	_, err = m.db.Exec(`
		UPDATE jobs
		SET
			status = 'failed',
			error_message = 'Maximum retry attempts exceeded after server restart',
			completed_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE status = 'processing'
			AND retry_count >= max_retries
			AND max_retries >= 0
	`)

	if err != nil {
		logger.Error("Failed to mark jobs as permanently failed: %v", err)
	}
}

// handleSuccessfulJob updates job and history records for a successful job.
func (w *Worker) handleSuccessfulJob(job *Job, result *JobResult, durationMs int) {
	logger.Info("Job %s completed successfully in %d ms", job.UID, durationMs)

	// Start a transaction
	tx, err := w.manager.db.Begin()
	if err != nil {
		logger.Error("Failed to begin transaction for successful job %s: %v", job.UID, err)
		return
	}

	var committed bool
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				logger.Error("Failed to rollback transaction: %v", rollbackErr)
			}
			logger.Error("Failed to update successful job %s: %v", job.UID, err)
		}
	}()

	// Update the job record
	now := time.Now()
	_, err = tx.Exec(`
		UPDATE jobs
		SET
			status = 'completed',
			completed_at = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE uid = $2
	`, now, job.UID)

	if err != nil {
		return
	}

	// Update the job history record
	var outputJSON []byte
	if result.Output != nil {
		outputJSON, err = json.Marshal(result.Output)
		if err != nil {
			outputJSON = []byte("{}")
			logger.Error("Failed to marshal job output for %s: %v", job.UID, err)
		}
	} else {
		outputJSON = []byte("{}")
	}

	_, err = tx.Exec(`
		UPDATE job_history
		SET
			status = 'completed',
			completed_at = $1,
			duration_ms = $2,
			output = $3
		WHERE uid = (
			SELECT uid
			FROM job_history
			WHERE job_uid = $4 AND status = 'started'
			ORDER BY created_at DESC
			LIMIT 1
		)
	`,
		now, durationMs, outputJSON, job.UID,
	)

	if err != nil {
		return
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction for successful job %s: %v", job.UID, err)
		return
	}
	committed = true
}

// handleFailedJob updates job and history records for a failed job.
func (w *Worker) handleFailedJob(job *Job, result *JobResult, durationMs int) {
	// Check if we should retry
	shouldRetry := job.RetryCount < job.MaxRetries

	logger.Error("Job %s failed: %s (attempt %d/%d)",
		job.UID, result.Error, job.RetryCount+1, job.MaxRetries)

	// Start a transaction
	tx, err := w.manager.db.Begin()
	if err != nil {
		logger.Error("Failed to begin transaction for failed job %s: %v", job.UID, err)
		return
	}

	var committed bool
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				logger.Error("Failed to rollback transaction: %v", rollbackErr)
			}
			logger.Error("Failed to update failed job %s: %v", job.UID, err)
		}
	}()

	// Update job history record
	var errorDetailsJSON []byte
	if result.ErrorDetails != nil {
		errorDetailsJSON, err = json.Marshal(result.ErrorDetails)
		if err != nil {
			errorDetailsJSON = []byte("{}")
			logger.Error("Failed to marshal error details for job %s: %v", job.UID, err)
		}
	} else {
		errorDetailsJSON = []byte("{}")
	}

	// Update the latest history entry
	now := time.Now()

	statusValue := "retry"
	if !shouldRetry {
		statusValue = "failed"
	}

	_, err = tx.Exec(`
		UPDATE job_history
		SET
			status = $1,
			completed_at = $2,
			duration_ms = $3,
			error_message = $4,
			error_details = $5
		WHERE uid = (
			SELECT uid
			FROM job_history
			WHERE job_uid = $6 AND status = 'started'
			ORDER BY created_at DESC
			LIMIT 1
		)
	`,
		statusValue, now, durationMs,
		result.Error, errorDetailsJSON, job.UID,
	)

	if err != nil {
		return
	}

	if shouldRetry {
		// Schedule job for retry
		nextRetry := time.Now().Add(time.Duration(w.manager.config.RetryDelay) * time.Second)
		errorMsg := result.Error

		_, err = tx.Exec(`
			UPDATE jobs
			SET
				status = 'pending',
				retry_count = $1,
				next_retry_at = $2,
				error_message = $3,
				worker_id = NULL,
				updated_at = CURRENT_TIMESTAMP
			WHERE uid = $4
		`, job.RetryCount+1, nextRetry, errorMsg, job.UID)

		if err != nil {
			return
		}

		logger.Info("Job %s will be retried in %d seconds (attempt %d/%d)",
			job.UID, w.manager.config.RetryDelay, job.RetryCount+1, job.MaxRetries)
	} else {
		// Mark job as permanently failed
		errorMsg := result.Error

		_, err = tx.Exec(`
			UPDATE jobs
			SET
				status = 'failed',
				error_message = $1,
				completed_at = $2,
				updated_at = CURRENT_TIMESTAMP
			WHERE uid = $3
		`, errorMsg, now, job.UID)

		if err != nil {
			return
		}

		logger.Error("Job %s failed permanently after %d attempts",
			job.UID, job.RetryCount+1)
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction for failed job %s: %v", job.UID, err)
	}
}
