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

package server

import (
	"testing"
	"time"
)

// TestSessionAffinityManager_CreateSession tests session creation.
func TestSessionAffinityManager_CreateSession(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")
	if sam == nil {
		t.Fatal("Expected session affinity manager to be created")
	}

	// Create a session
	session := sam.CreateSession("user-123")
	if session == nil {
		t.Fatal("Expected session to be created")
	}

	if session.SessionID == "" {
		t.Error("Expected session ID to be generated")
	}

	if session.InstanceID != "instance-1" {
		t.Errorf("Expected instance ID to be 'instance-1', got %s", session.InstanceID)
	}

	if session.UserID != "user-123" {
		t.Errorf("Expected user ID to be 'user-123', got %s", session.UserID)
	}

	if !session.Active {
		t.Error("Expected session to be active")
	}
}

// TestSessionAffinityManager_GetSession tests session retrieval.
func TestSessionAffinityManager_GetSession(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Create a session
	session := sam.CreateSession("user-123")

	// Retrieve the session
	retrievedSession, exists := sam.GetSession(session.SessionID)
	if !exists {
		t.Error("Expected session to exist")
	}

	if retrievedSession.SessionID != session.SessionID {
		t.Error("Retrieved session ID doesn't match")
	}

	// Try to retrieve non-existent session
	_, exists = sam.GetSession("non-existent")
	if exists {
		t.Error("Expected non-existent session to not exist")
	}
}

// TestSessionAffinityManager_IsSessionAffine tests session affinity check.
func TestSessionAffinityManager_IsSessionAffine(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")
	sam2 := NewSessionAffinityManager("instance-2")

	// Create session on instance 1
	session := sam.CreateSession("user-123")

	// Check affinity on instance 1
	if !sam.IsSessionAffine(session.SessionID) {
		t.Error("Expected session to be affine to instance 1")
	}

	// Check affinity on instance 2 (should be false)
	if sam2.IsSessionAffine(session.SessionID) {
		t.Error("Expected session to not be affine to instance 2")
	}
}

// TestSessionAffinityManager_UpdateLastSeen tests last seen update.
func TestSessionAffinityManager_UpdateLastSeen(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Create a session
	session := sam.CreateSession("user-123")
	originalLastSeen := session.LastSeen

	// Wait a bit and update last seen
	time.Sleep(10 * time.Millisecond)
	sam.UpdateLastSeen(session.SessionID)

	// Get the updated session
	updatedSession, exists := sam.GetSession(session.SessionID)
	if !exists {
		t.Fatal("Expected session to exist")
	}

	if !updatedSession.LastSeen.After(originalLastSeen) {
		t.Error("Expected last seen to be updated")
	}
}

// TestSessionAffinityManager_RemoveSession tests session removal.
func TestSessionAffinityManager_RemoveSession(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Create a session
	session := sam.CreateSession("user-123")

	// Verify session exists
	_, exists := sam.GetSession(session.SessionID)
	if !exists {
		t.Fatal("Expected session to exist before removal")
	}

	// Remove the session
	sam.RemoveSession(session.SessionID)

	// Verify session is removed
	_, exists = sam.GetSession(session.SessionID)
	if exists {
		t.Error("Expected session to be removed")
	}
}

// TestSessionAffinityManager_GetActiveSessionCount tests session counting.
func TestSessionAffinityManager_GetActiveSessionCount(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Initially should be 0
	if count := sam.GetActiveSessionCount(); count != 0 {
		t.Errorf("Expected 0 active sessions, got %d", count)
	}

	// Create some sessions
	sam.CreateSession("user-1")
	sam.CreateSession("user-2")
	sam.CreateSession("user-3")

	// Should be 3
	if count := sam.GetActiveSessionCount(); count != 3 {
		t.Errorf("Expected 3 active sessions, got %d", count)
	}

	// Remove one session
	sessions := sam.GetUserSessions("user-1")
	if len(sessions) > 0 {
		sam.RemoveSession(sessions[0].SessionID)
	}

	// Should be 2
	if count := sam.GetActiveSessionCount(); count != 2 {
		t.Errorf("Expected 2 active sessions, got %d", count)
	}
}

// TestSessionAffinityManager_GetUserSessions tests user session retrieval.
func TestSessionAffinityManager_GetUserSessions(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Create multiple sessions for same user
	sam.CreateSession("user-123")
	sam.CreateSession("user-123")
	sam.CreateSession("user-456")

	// Get sessions for user-123
	sessions := sam.GetUserSessions("user-123")
	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions for user-123, got %d", len(sessions))
	}

	// Get sessions for user-456
	sessions = sam.GetUserSessions("user-456")
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session for user-456, got %d", len(sessions))
	}

	// Get sessions for non-existent user
	sessions = sam.GetUserSessions("non-existent")
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions for non-existent user, got %d", len(sessions))
	}
}

// TestSessionAffinityManager_ShouldHandleRequest tests request handling logic.
func TestSessionAffinityManager_ShouldHandleRequest(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Request without session ID should be handled
	req := &SessionRequest{
		SessionID: "",
		UserID:    "user-123",
		Path:      "/api/test",
	}

	if !sam.ShouldHandleRequest(req) {
		t.Error("Expected request without session ID to be handled")
	}

	// Create a session
	session := sam.CreateSession("user-123")

	// Request with local session ID should be handled
	req.SessionID = session.SessionID
	if !sam.ShouldHandleRequest(req) {
		t.Error("Expected request with local session ID to be handled")
	}

	// Request with non-existent session ID should not be handled
	req.SessionID = "non-existent-session"
	if sam.ShouldHandleRequest(req) {
		t.Error("Expected request with non-existent session ID to not be handled")
	}
}

// TestSessionAffinityManager_GetStickyCookie tests sticky cookie generation.
func TestSessionAffinityManager_GetStickyCookie(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	cookie := sam.GetStickyCookie("session-123")
	expected := "instance-1.session-123"

	if cookie != expected {
		t.Errorf("Expected sticky cookie '%s', got '%s'", expected, cookie)
	}
}

// TestSessionAffinityManager_ParseStickyCookie tests sticky cookie parsing.
func TestSessionAffinityManager_ParseStickyCookie(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Test valid cookie
	instanceID, sessionID := sam.ParseStickyCookie("instance-1.session-123")
	if instanceID != "instance-1" {
		t.Errorf("Expected instance ID 'instance-1', got '%s'", instanceID)
	}
	if sessionID != "session-123" {
		t.Errorf("Expected session ID 'session-123', got '%s'", sessionID)
	}

	// Test cookie without instance ID
	instanceID, sessionID = sam.ParseStickyCookie("session-only")
	if instanceID != "" {
		t.Errorf("Expected empty instance ID, got '%s'", instanceID)
	}
	if sessionID != "session-only" {
		t.Errorf("Expected session ID 'session-only', got '%s'", sessionID)
	}
}

// TestSessionAffinityManager_GetInstanceLoad tests load information.
func TestSessionAffinityManager_GetInstanceLoad(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Create some sessions
	sam.CreateSession("user-1")
	sam.CreateSession("user-2")

	load := sam.GetInstanceLoad()

	if load["instance_id"] != "instance-1" {
		t.Errorf("Expected instance_id 'instance-1', got %v", load["instance_id"])
	}

	if load["active_sessions"] != 2 {
		t.Errorf("Expected active_sessions 2, got %v", load["active_sessions"])
	}

	if load["total_sessions"] != 2 {
		t.Errorf("Expected total_sessions 2, got %v", load["total_sessions"])
	}
}

// TestSessionAffinityManager_CleanupExpiredSessions tests session cleanup.
func TestSessionAffinityManager_CleanupExpiredSessions(t *testing.T) {
	sam := NewSessionAffinityManager("instance-1")

	// Create a session
	session := sam.CreateSession("user-123")

	// Manually set LastSeen to a past time
	sam.mutex.Lock()
	sam.sessions[session.SessionID].LastSeen = time.Now().Add(-10 * time.Minute)
	sam.mutex.Unlock()

	// Run cleanup with 5-minute max idle time
	sam.cleanupExpiredSessions(5 * time.Minute)

	// Session should be removed
	_, exists := sam.GetSession(session.SessionID)
	if exists {
		t.Error("Expected expired session to be cleaned up")
	}

	// Active session count should be 0
	if count := sam.GetActiveSessionCount(); count != 0 {
		t.Errorf("Expected 0 active sessions after cleanup, got %d", count)
	}
}
