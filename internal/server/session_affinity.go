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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/daison12006013/turboscript/internal/logger"
)

// SessionAffinityManager manages sticky sessions for horizontal scaling.
type SessionAffinityManager struct {
	sessions    map[string]*SessionInfo
	instanceID  string
	mutex       sync.RWMutex
	cleanupTick time.Duration
	stopCleanup chan bool
}

// SessionInfo holds session affinity data.
type SessionInfo struct {
	SessionID  string    `json:"session_id"`
	InstanceID string    `json:"instance_id"`
	UserID     string    `json:"user_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeen   time.Time `json:"last_seen"`
	Active     bool      `json:"active"`
}

// NewSessionAffinityManager creates a new session affinity manager.
func NewSessionAffinityManager(instanceID string) *SessionAffinityManager {
	return &SessionAffinityManager{
		sessions:    make(map[string]*SessionInfo),
		instanceID:  instanceID,
		cleanupTick: 5 * time.Minute,
		stopCleanup: make(chan bool),
	}
}

// GenerateSessionID creates a new unique session ID.
func (sam *SessionAffinityManager) GenerateSessionID() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		logger.Error("Failed to generate random bytes for session ID: %v", err)
		// Fallback to timestamp-based ID
		return fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// CreateSession creates a new session with affinity to this instance.
func (sam *SessionAffinityManager) CreateSession(userID string) *SessionInfo {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	sessionID := sam.GenerateSessionID()
	now := time.Now()

	session := &SessionInfo{
		SessionID:  sessionID,
		InstanceID: sam.instanceID,
		UserID:     userID,
		CreatedAt:  now,
		LastSeen:   now,
		Active:     true,
	}

	sam.sessions[sessionID] = session
	return session
}

// GetSession retrieves session information.
func (sam *SessionAffinityManager) GetSession(sessionID string) (*SessionInfo, bool) {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	session, exists := sam.sessions[sessionID]
	if !exists || !session.Active {
		return nil, false
	}

	return session, true
}

// UpdateLastSeen updates the last seen timestamp for a session.
func (sam *SessionAffinityManager) UpdateLastSeen(sessionID string) {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	if session, exists := sam.sessions[sessionID]; exists && session.Active {
		session.LastSeen = time.Now()
	}
}

// IsSessionAffine checks if a session belongs to this instance.
func (sam *SessionAffinityManager) IsSessionAffine(sessionID string) bool {
	session, exists := sam.GetSession(sessionID)
	if !exists {
		return false
	}

	return session.InstanceID == sam.instanceID
}

// RemoveSession removes a session.
func (sam *SessionAffinityManager) RemoveSession(sessionID string) {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	if session, exists := sam.sessions[sessionID]; exists {
		session.Active = false
		delete(sam.sessions, sessionID)
	}
}

// GetActiveSessionCount returns the number of active sessions on this instance.
func (sam *SessionAffinityManager) GetActiveSessionCount() int {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	count := 0
	for _, session := range sam.sessions {
		if session.Active {
			count++
		}
	}
	return count
}

// GetUserSessions returns all active sessions for a specific user.
func (sam *SessionAffinityManager) GetUserSessions(userID string) []*SessionInfo {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	var userSessions []*SessionInfo
	for _, session := range sam.sessions {
		if session.Active && session.UserID == userID {
			userSessions = append(userSessions, session)
		}
	}
	return userSessions
}

// StartCleanup starts the cleanup routine for expired sessions.
func (sam *SessionAffinityManager) StartCleanup(maxIdleTime time.Duration) {
	go func() {
		ticker := time.NewTicker(sam.cleanupTick)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				sam.cleanupExpiredSessions(maxIdleTime)
			case <-sam.stopCleanup:
				return
			}
		}
	}()
}

// StopCleanup stops the cleanup routine.
func (sam *SessionAffinityManager) StopCleanup() {
	select {
	case sam.stopCleanup <- true:
	default:
	}
}

// cleanupExpiredSessions removes sessions that haven't been seen for too long.
func (sam *SessionAffinityManager) cleanupExpiredSessions(maxIdleTime time.Duration) {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	now := time.Now()
	var expiredSessions []string

	for sessionID, session := range sam.sessions {
		if session.Active && now.Sub(session.LastSeen) > maxIdleTime {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	for _, sessionID := range expiredSessions {
		if session, exists := sam.sessions[sessionID]; exists {
			session.Active = false
			delete(sam.sessions, sessionID)
		}
	}
}

// GetInstanceLoad returns load information for this instance.
func (sam *SessionAffinityManager) GetInstanceLoad() map[string]interface{} {
	sam.mutex.RLock()
	defer sam.mutex.RUnlock()

	return map[string]interface{}{
		"instance_id":     sam.instanceID,
		"active_sessions": sam.GetActiveSessionCount(),
		"total_sessions":  len(sam.sessions),
		"memory_sessions": len(sam.sessions),
		"timestamp":       time.Now(),
	}
}

// TransferSession marks a session for transfer to another instance.
func (sam *SessionAffinityManager) TransferSession(sessionID, targetInstanceID string) bool {
	sam.mutex.Lock()
	defer sam.mutex.Unlock()

	session, exists := sam.sessions[sessionID]
	if !exists || !session.Active {
		return false
	}

	// Mark session as transferred
	session.InstanceID = targetInstanceID
	session.Active = false
	delete(sam.sessions, sessionID)

	return true
}

// ValidateSessionRequest checks if a request should be handled by this instance.
type SessionRequest struct {
	SessionID string
	UserID    string
	Path      string
	Headers   map[string]string
}

// ShouldHandleRequest determines if this instance should handle the request.
func (sam *SessionAffinityManager) ShouldHandleRequest(req *SessionRequest) bool {
	// If no session ID provided, this instance can handle it
	if req.SessionID == "" {
		return true
	}

	// Check if session is affine to this instance
	return sam.IsSessionAffine(req.SessionID)
}

// GetStickyCookie generates a sticky session cookie value.
func (sam *SessionAffinityManager) GetStickyCookie(sessionID string) string {
	return sam.instanceID + "." + sessionID
}

// ParseStickyCookie parses a sticky session cookie to extract instance and session.
func (sam *SessionAffinityManager) ParseStickyCookie(cookie string) (instanceID, sessionID string) {
	// Simple format: instanceID.sessionID
	parts := make([]string, 0, 2)

	// Find the first dot
	dotIndex := -1
	for i, r := range cookie {
		if r == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex == -1 {
		return "", cookie // No instance ID found
	}

	parts = append(parts, cookie[:dotIndex])
	parts = append(parts, cookie[dotIndex+1:])

	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	return "", cookie
}
