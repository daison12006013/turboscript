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

package tsengine

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/dop251/goja"
)

// MockJobManager is a mock implementation of JobManager interface.
type MockJobManager struct {
	DispatchJobFn    func(jobPath string, payload map[string]any) error
	DispatchJobCalls []struct {
		JobPath string
		Payload map[string]any
	}
}

func (m *MockJobManager) DispatchJob(jobPath string, payload map[string]any) error {
	if m.DispatchJobFn == nil {
		return nil
	}
	m.DispatchJobCalls = append(m.DispatchJobCalls, struct {
		JobPath string
		Payload map[string]any
	}{jobPath, payload})
	return m.DispatchJobFn(jobPath, payload)
}

// MockDBJobManager is a mock implementation of DBJobManager interface.
type MockDBJobManager struct {
	DispatchJobFn           func(jobPath string, payload map[string]any) error
	DispatchJobWithReturnFn func(jobPath string, payload map[string]any) (string, error)
	DispatchJobCalls        []struct {
		JobPath string
		Payload map[string]any
	}
	DispatchJobWithReturnCalls []struct {
		JobPath string
		Payload map[string]any
	}
}

func (m *MockDBJobManager) DispatchJob(jobPath string, payload map[string]any) error {
	if m.DispatchJobFn == nil {
		return nil
	}
	m.DispatchJobCalls = append(m.DispatchJobCalls, struct {
		JobPath string
		Payload map[string]any
	}{jobPath, payload})
	return m.DispatchJobFn(jobPath, payload)
}

func (m *MockDBJobManager) DispatchJobWithReturn(jobPath string, payload map[string]any) (string, error) {
	if m.DispatchJobWithReturnFn == nil {
		return "", nil
	}
	m.DispatchJobWithReturnCalls = append(m.DispatchJobWithReturnCalls, struct {
		JobPath string
		Payload map[string]any
	}{jobPath, payload})
	return m.DispatchJobWithReturnFn(jobPath, payload)
}

func TestNewTurboJobUtils(t *testing.T) {
	// Create mock job manager
	mockJobManager := &MockJobManager{}

	// Create turbo job utils instance
	turboJobUtils := NewTurboJobUtils(mockJobManager)

	// Assert that job manager is set correctly
	if turboJobUtils == nil {
		t.Fatal("NewTurboJobUtils() returned nil")
	}

	if turboJobUtils.jobManager != mockJobManager {
		t.Errorf("Expected jobManager to be %v, got %v", mockJobManager, turboJobUtils.jobManager)
	}
}

func TestExecuteJob_BasicJobManager(t *testing.T) {
	// Create mock job manager
	mockJobManager := &MockJobManager{}

	// Setup expected behavior
	mockJobManager.DispatchJobFn = func(jobPath string, payload map[string]any) error {
		return nil
	}

	// Create goja runtime and turbo job utils
	rt := goja.New()
	turboJobUtils := NewTurboJobUtils(mockJobManager)

	// Create function call with arguments
	jobPath := rt.ToValue("test-job")
	payload := rt.ToValue(map[string]any{
		"user_id": 123,
		"action":  "test",
	})
	args := []goja.Value{jobPath, payload}
	call := goja.FunctionCall{This: rt.GlobalObject(), Arguments: args}

	// Test the function call
	result := turboJobUtils.ExecuteJob(call, rt)

	// Verify the call was made correctly
	if len(mockJobManager.DispatchJobCalls) != 1 {
		t.Errorf("Expected 1 call to DispatchJob, got %d", len(mockJobManager.DispatchJobCalls))
	}

	if len(mockJobManager.DispatchJobCalls) > 0 {
		callDetails := mockJobManager.DispatchJobCalls[0]
		if callDetails.JobPath != "test-job" {
			t.Errorf("Expected jobPath to be %s, got %s", "test-job", callDetails.JobPath)
		}
	}

	if result.String() != "" {
		t.Errorf("Expected result to be empty string, got %s", result.String())
	}
}

func TestExecuteJob_DBJobManager(t *testing.T) {
	// Create mock DB job manager
	jobUID := "job-1234-5678-9abc"
	mockDBJobManager := &MockDBJobManager{
		DispatchJobWithReturnFn: func(jobPath string, payload map[string]any) (string, error) {
			return jobUID, nil
		},
	}

	// Create goja runtime and turbo job utils
	rt := goja.New()
	turboJobUtils := NewTurboJobUtils(mockDBJobManager)

	// Create function call with arguments
	jobPath := rt.ToValue("email-job")
	payload := rt.ToValue(map[string]any{
		"email":    "test@example.com",
		"template": "welcome",
	})
	args := []goja.Value{jobPath, payload}
	call := goja.FunctionCall{This: rt.GlobalObject(), Arguments: args}

	// Test the function call
	result := turboJobUtils.ExecuteJob(call, rt)

	// Verify the call was made correctly
	if len(mockDBJobManager.DispatchJobWithReturnCalls) != 1 {
		t.Errorf("Expected 1 call to DispatchJobWithReturn, got %d", len(mockDBJobManager.DispatchJobWithReturnCalls))
	}

	if len(mockDBJobManager.DispatchJobWithReturnCalls) > 0 {
		callDetails := mockDBJobManager.DispatchJobWithReturnCalls[0]
		if callDetails.JobPath != "email-job" {
			t.Errorf("Expected jobPath to be %s, got %s", "email-job", callDetails.JobPath)
		}
	}

	if result.String() != jobUID {
		t.Errorf("Expected result to be %s, got %s", jobUID, result.String())
	}
}

func TestExecuteJob_Error(t *testing.T) {
	// Create mock job manager
	expectedErr := errors.New("job dispatch failed")
	mockJobManager := &MockJobManager{
		DispatchJobFn: func(jobPath string, payload map[string]any) error {
			return expectedErr
		},
	}

	// Create goja runtime and turbo job utils
	rt := goja.New()
	turboJobUtils := NewTurboJobUtils(mockJobManager)

	// Create function call with arguments
	jobPath := rt.ToValue("failing-job")
	payload := rt.ToValue(map[string]any{
		"data": "will fail",
	})
	args := []goja.Value{jobPath, payload}
	call := goja.FunctionCall{This: rt.GlobalObject(), Arguments: args}

	// Test the function call - expect panic with Go error
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected function to panic, but it did not")
			return
		}

		// Handle different types of panic values
		var errorMsg string
		switch v := r.(type) {
		case *goja.Exception:
			errorMsg = v.Error()
		case *goja.Object:
			errorMsg = v.String()
		case error:
			errorMsg = v.Error()
		default:
			errorMsg = fmt.Sprintf("%v", v)
		}

		expectedMsg := "failed to dispatch job: job dispatch failed"
		if !strings.Contains(errorMsg, expectedMsg) {
			t.Errorf("Expected error to contain %q, got %q", expectedMsg, errorMsg)
		}
	}()

	turboJobUtils.ExecuteJob(call, rt)
	t.Error("Expected function to panic, but it did not")
}

func TestExecuteJob_MissingArguments(t *testing.T) {
	// Create mock job manager
	mockJobManager := &MockJobManager{}

	// Create goja runtime and turbo job utils
	rt := goja.New()
	turboJobUtils := NewTurboJobUtils(mockJobManager)

	// Create function call with insufficient arguments
	args := []goja.Value{rt.ToValue("test-job")} // Missing payload
	call := goja.FunctionCall{This: rt.GlobalObject(), Arguments: args}

	// Test the function call - expect panic with Go error
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected function to panic, but it did not")
			return
		}

		// Handle different types of panic values
		var errorMsg string
		switch v := r.(type) {
		case *goja.Exception:
			errorMsg = v.Error()
		case *goja.Object:
			errorMsg = v.String()
		case error:
			errorMsg = v.Error()
		default:
			errorMsg = fmt.Sprintf("%v", v)
		}

		expectedMsg := "turboJob requires 2 arguments"
		if !strings.Contains(errorMsg, expectedMsg) {
			t.Errorf("Expected error to contain %q, got %q", expectedMsg, errorMsg)
		}
	}()

	turboJobUtils.ExecuteJob(call, rt)
	t.Error("Expected function to panic, but it did not")
}

func TestExecuteJob_EmptyPath(t *testing.T) {
	// Create mock job manager
	mockJobManager := &MockJobManager{}

	// Create goja runtime and turbo job utils
	rt := goja.New()
	turboJobUtils := NewTurboJobUtils(mockJobManager)

	// Create function call with empty job path
	jobPath := rt.ToValue("")
	payload := rt.ToValue(map[string]any{})
	args := []goja.Value{jobPath, payload}
	call := goja.FunctionCall{This: rt.GlobalObject(), Arguments: args}

	// Test the function call - expect panic with Go error
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected function to panic, but it did not")
			return
		}

		// Handle different types of panic values
		var errorMsg string
		switch v := r.(type) {
		case *goja.Exception:
			errorMsg = v.Error()
		case *goja.Object:
			errorMsg = v.String()
		case error:
			errorMsg = v.Error()
		default:
			errorMsg = fmt.Sprintf("%v", v)
		}

		expectedMsg := "job path cannot be empty"
		if !strings.Contains(errorMsg, expectedMsg) {
			t.Errorf("Expected error to contain %q, got %q", expectedMsg, errorMsg)
		}
	}()

	turboJobUtils.ExecuteJob(call, rt)
	t.Error("Expected function to panic, but it did not")
}

func TestExecuteJob_InvalidPayload(t *testing.T) {
	// Create mock job manager
	mockJobManager := &MockJobManager{}

	// Create goja runtime and turbo job utils
	rt := goja.New()
	turboJobUtils := NewTurboJobUtils(mockJobManager)

	// Create function call with invalid payload (string instead of object)
	jobPath := rt.ToValue("test-job")
	payload := rt.ToValue("not-an-object")
	args := []goja.Value{jobPath, payload}
	call := goja.FunctionCall{This: rt.GlobalObject(), Arguments: args}

	// Test the function call - expect panic with Go error
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected function to panic, but it did not")
			return
		}

		// Handle different types of panic values
		var errorMsg string
		switch v := r.(type) {
		case *goja.Exception:
			errorMsg = v.Error()
		case *goja.Object:
			errorMsg = v.String()
		case error:
			errorMsg = v.Error()
		default:
			errorMsg = fmt.Sprintf("%v", v)
		}

		expectedMsg := "payload must be an object"
		if !strings.Contains(errorMsg, expectedMsg) {
			t.Errorf("Expected error to contain %q, got %q", expectedMsg, errorMsg)
		}
	}()

	turboJobUtils.ExecuteJob(call, rt)
	t.Error("Expected function to panic, but it did not")
}

func TestExecuteJob_NilJobManager(t *testing.T) {
	// Create turbo job utils with nil job manager
	rt := goja.New()
	turboJobUtils := NewTurboJobUtils(nil)

	// Create function call with arguments
	jobPath := rt.ToValue("test-job")
	payload := rt.ToValue(map[string]any{})
	args := []goja.Value{jobPath, payload}
	call := goja.FunctionCall{This: rt.GlobalObject(), Arguments: args}

	// Test the function call - expect panic with Go error
	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected function to panic, but it did not")
			return
		}

		// Handle different types of panic values
		var errorMsg string
		switch v := r.(type) {
		case *goja.Exception:
			errorMsg = v.Error()
		case *goja.Object:
			errorMsg = v.String()
		case error:
			errorMsg = v.Error()
		default:
			errorMsg = fmt.Sprintf("%v", v)
		}

		expectedMsg := "job manager is not initialized"
		if !strings.Contains(errorMsg, expectedMsg) {
			t.Errorf("Expected error to contain %q, got %q", expectedMsg, errorMsg)
		}
	}()

	turboJobUtils.ExecuteJob(call, rt)
	t.Error("Expected function to panic, but it did not")
}

func TestGetMapKeys(t *testing.T) {
	testMap := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	keys := getMapKeys(testMap)

	// Check if all keys are present (order may vary)
	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// Check that each key is present
	keyFound := map[string]bool{
		"key1": false,
		"key2": false,
		"key3": false,
	}

	for _, key := range keys {
		keyFound[key] = true
	}

	for key, found := range keyFound {
		if !found {
			t.Errorf("Expected key %q to be in result, but it wasn't", key)
		}
	}
}
