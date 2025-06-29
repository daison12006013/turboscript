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
	"github.com/daison12006013/turboscript/internal/logger"
	"github.com/daison12006013/turboscript/internal/tsengine"
)

// getExecutor gets an isolated executor from the pool.
// If the pool is empty, it creates a new one temporarily.
func (s *Server) getExecutor() *tsengine.TSExecutor {
	select {
	case executor := <-s.executorPool:
		return executor
	default:
		// Pool is empty, create a temporary isolated executor
		logger.Debug("Executor pool exhausted, creating temporary executor")
		fileResolver := tsengine.GetResolverFromConfig(s.cfg.PreferTS, s.cfg.PreferJS)
		tempExecutor := tsengine.NewIsolatedTSExecutorWithResolverAndConfig(s.cfg.PreserveResponse, fileResolver, &s.cfg.TypeScript)

		// Set database manager if available
		if s.dbManager != nil {
			tempExecutor.SetDatabaseManager(s.dbManager)

			// Also set the default database for backward compatibility
			if defaultDB, err := s.dbManager.GetDefaultConnection(); err == nil {
				tempExecutor.SetDatabase(defaultDB)
			}
		}

		// Set cache configuration for turboCache operations
		tempExecutor.SetCacheConfig(&s.cfg.Cache)

		// Set markdown base path to app/routes for static file access
		tempExecutor.SetMarkdownBasePath("app/routes")

		// Set up job manager for temporary executor if available
		if s.jobManager != nil {
			if jm, ok := s.jobManager.(interface {
				DispatchJob(string, map[string]any) error
			}); ok {
				tempExecutor.SetJobManager(jm)
			}
		}

		// Set up email service for temporary executor if available
		if s.emailService != nil {
			tempExecutor.SetEmailService(s.emailService)
		}

		return tempExecutor
	}
}

// returnExecutor returns an executor to the pool.
// If the pool is full, the executor is discarded (will be garbage collected).
func (s *Server) returnExecutor(executor *tsengine.TSExecutor) {
	select {
	case s.executorPool <- executor:
		// Successfully returned to pool
	default:
		// Pool is full, terminate the executor and let it be garbage collected
		executor.TerminateAsync()
	}
}
