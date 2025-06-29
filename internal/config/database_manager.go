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

package config

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/daison12006013/turboscript/internal/logger"
	// Import PostgreSQL driver for database connectivity.
	_ "github.com/lib/pq"
)

// DatabaseManager manages multiple database connections.
type DatabaseManager struct {
	connections map[string]*sql.DB
	config      *DatabaseConfig
	mutex       sync.RWMutex
}

// NewDatabaseManager creates a new database manager with the given configuration.
func NewDatabaseManager(config *DatabaseConfig) *DatabaseManager {
	return &DatabaseManager{
		connections: make(map[string]*sql.DB),
		config:      config,
	}
}

// InitializeConnections establishes all configured database connections.
func (dm *DatabaseManager) InitializeConnections() error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	for name, connConfig := range dm.config.Connections {
		connectionString, err := connConfig.BuildConnectionString()
		if err != nil {
			return fmt.Errorf("failed to build connection string for '%s': %w", name, err)
		}

		logger.Info("Connecting to database: %s", name)
		db, err := sql.Open("postgres", connectionString)
		if err != nil {
			return fmt.Errorf("failed to open connection '%s': %w", name, err)
		}

		// Test the connection
		if err := db.Ping(); err != nil {
			if closeErr := db.Close(); closeErr != nil {
				logger.Error("Failed to close database connection after ping failure: %v", closeErr)
			}
			return fmt.Errorf("failed to ping connection '%s': %w", name, err)
		}

		// Configure connection pool settings
		if connConfig.MaxOpenConnections > 0 {
			db.SetMaxOpenConns(connConfig.MaxOpenConnections)
		}
		if connConfig.MaxIdleConnections > 0 {
			db.SetMaxIdleConns(connConfig.MaxIdleConnections)
		}

		dm.connections[name] = db
		logger.Debug("Successfully connected to database: %s", name)
	}

	return nil
}

// getConnection returns a database connection by name.
// If name is empty, returns the default connection.
func (dm *DatabaseManager) getConnection(name string) (*sql.DB, error) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	// Use default connection if name is empty
	if name == "" {
		name = dm.config.Default
	}

	db, exists := dm.connections[name]
	if !exists {
		return nil, fmt.Errorf("database connection '%s' not found", name)
	}

	return db, nil
}

// GetDefaultConnection returns the default database connection.
func (dm *DatabaseManager) GetDefaultConnection() (*sql.DB, error) {
	return dm.getConnection("")
}

// Close closes all database connections.
func (dm *DatabaseManager) Close() error {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	var lastErr error
	for name, db := range dm.connections {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection '%s': %v", name, err)
			lastErr = err
		} else {
			logger.Debug("Closed database connection: %s", name)
		}
	}

	dm.connections = make(map[string]*sql.DB)
	return lastErr
}
