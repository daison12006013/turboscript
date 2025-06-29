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

package pg

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dop251/goja"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// EventLoopRunner represents the interface for running functions on the event loop.
type EventLoopRunner interface {
	RunOnLoop(fn func(*goja.Runtime)) bool
}

// PgModule represents the pg module for goja.
type PgModule struct {
	runtime *goja.Runtime
	loop    EventLoopRunner
}

// Client represents a PostgreSQL client connection.
type Client struct {
	db      *sql.DB
	runtime *goja.Runtime
	loop    EventLoopRunner
	config  *ConnectionConfig
}

// Pool represents a PostgreSQL connection pool.
type Pool struct {
	db      *sql.DB
	runtime *goja.Runtime
	loop    EventLoopRunner
	config  *ConnectionConfig
}

// ConnectionConfig represents the configuration for a PostgreSQL connection.
type ConnectionConfig struct {
	Host                string `json:"host"`
	Port                int    `json:"port"`
	Database            string `json:"database"`
	User                string `json:"user"`
	Password            string `json:"password"`
	ConnectionString    string `json:"connectionString"`
	SSL                 string `json:"ssl"`
	SSLMode             string `json:"sslmode"`
	ConnectionTimeoutMs int    `json:"connectionTimeoutMillis"`
	QueryTimeoutMs      int    `json:"queryTimeoutMillis"`
	IdleTimeoutMs       int    `json:"idleTimeoutMillis"`
	MaxConnections      int    `json:"max"`
	IdleConnections     int    `json:"idleCount"`
	StatementTimeoutMs  int    `json:"statement_timeout"`
	LockTimeoutMs       int    `json:"lock_timeout"`
	IdleInTransactionMs int    `json:"idle_in_transaction_session_timeout"`
	ApplicationName     string `json:"application_name"`
}

// QueryResult represents the result of a query.
type QueryResult struct {
	Rows         []map[string]interface{} `json:"rows"`
	RowCount     int                      `json:"rowCount"`
	Command      string                   `json:"command"`
	Fields       []Field                  `json:"fields"`
	ProcessingMs float64                  `json:"processingTimeMs"`
}

// Field represents a database field metadata.
type Field struct {
	Name         string `json:"name"`
	TableID      int    `json:"tableID"`
	ColumnID     int    `json:"columnID"`
	DataTypeID   int    `json:"dataTypeID"`
	DataTypeSize int    `json:"dataTypeSize"`
	TypeModifier int    `json:"typeModifier"`
	Format       string `json:"format"`
}

// New creates a new PostgreSQL module instance.
func New(runtime *goja.Runtime, loop EventLoopRunner) *PgModule {
	return &PgModule{
		runtime: runtime,
		loop:    loop,
	}
}

// Register registers the pg module with the goja runtime.
func (p *PgModule) Register() error {
	module := p.runtime.NewObject()

	// Client constructor
	if err := module.Set("Client", p.clientConstructor); err != nil {
		return fmt.Errorf("failed to set Client constructor: %w", err)
	}

	// Pool constructor
	if err := module.Set("Pool", p.poolConstructor); err != nil {
		return fmt.Errorf("failed to set Pool constructor: %w", err)
	}

	// Utility functions
	if err := module.Set("native", p.getNativeBindings); err != nil {
		return fmt.Errorf("failed to set native bindings: %w", err)
	}

	// Default export is Client constructor for compatibility
	if err := module.Set("default", p.clientConstructor); err != nil {
		return fmt.Errorf("failed to set default export: %w", err)
	}

	// Connection defaults
	defaults := p.runtime.NewObject()
	_ = defaults.Set("host", "localhost")
	_ = defaults.Set("port", 5432)
	_ = defaults.Set("database", "postgres")
	_ = defaults.Set("user", "postgres")
	_ = defaults.Set("password", "")
	_ = defaults.Set("ssl", false)
	_ = defaults.Set("sslmode", "prefer")
	_ = defaults.Set("connectionTimeoutMillis", 30000)
	_ = defaults.Set("queryTimeoutMillis", 30000)
	_ = defaults.Set("idleTimeoutMillis", 30000)
	_ = defaults.Set("max", 10)
	_ = defaults.Set("idleCount", 0)
	_ = defaults.Set("application_name", "goja-pg-client")

	if err := module.Set("defaults", defaults); err != nil {
		return fmt.Errorf("failed to set defaults: %w", err)
	}

	return p.runtime.Set("pg", module)
}

// clientConstructor creates a new PostgreSQL client.
func (p *PgModule) clientConstructor(call goja.ConstructorCall) *goja.Object {
	config := p.parseConfig(call.Arguments)
	client := &Client{
		runtime: p.runtime,
		loop:    p.loop,
		config:  config,
	}

	obj := call.This
	p.setupClientMethods(obj, client)
	return nil
}

// poolConstructor creates a new PostgreSQL connection pool.
func (p *PgModule) poolConstructor(call goja.ConstructorCall) *goja.Object {
	config := p.parseConfig(call.Arguments)
	pool := &Pool{
		runtime: p.runtime,
		loop:    p.loop,
		config:  config,
	}

	obj := call.This
	p.setupPoolMethods(obj, pool)
	return nil
}

// setupClientMethods sets up all client methods on the JavaScript object.
func (p *PgModule) setupClientMethods(obj *goja.Object, client *Client) {
	_ = obj.Set("connect", client.connect)
	_ = obj.Set("end", client.end)
	_ = obj.Set("query", client.query)
	_ = obj.Set("release", client.release)
	_ = obj.Set("on", client.on)
	_ = obj.Set("off", client.off)
	_ = obj.Set("removeListener", client.removeListener)
	_ = obj.Set("copyFrom", client.copyFrom)
	_ = obj.Set("copyTo", client.copyTo)
	_ = obj.Set("pauseDrain", client.pauseDrain)
	_ = obj.Set("resumeDrain", client.resumeDrain)
	_ = obj.Set("escapeIdentifier", client.escapeIdentifier)
	_ = obj.Set("escapeLiteral", client.escapeLiteral)
}

// setupPoolMethods sets up all pool methods on the JavaScript object.
func (p *PgModule) setupPoolMethods(obj *goja.Object, pool *Pool) {
	_ = obj.Set("connect", pool.connect)
	_ = obj.Set("end", pool.end)
	_ = obj.Set("query", pool.query)
	_ = obj.Set("on", pool.on)
	_ = obj.Set("off", pool.off)
	_ = obj.Set("removeListener", pool.removeListener)
	_ = obj.Set("totalCount", pool.totalCount)
	_ = obj.Set("idleCount", pool.idleCount)
	_ = obj.Set("waitingCount", pool.waitingCount)
}

// parseConfig parses the configuration from function call arguments.
func (p *PgModule) parseConfig(args []goja.Value) *ConnectionConfig {
	config := &ConnectionConfig{
		Host:                "localhost",
		Port:                5432,
		Database:            "postgres",
		User:                "postgres",
		Password:            "",
		SSL:                 "prefer",
		SSLMode:             "prefer",
		ConnectionTimeoutMs: 30000,
		QueryTimeoutMs:      30000,
		IdleTimeoutMs:       30000,
		MaxConnections:      10,
		IdleConnections:     0,
		ApplicationName:     "goja-pg-client",
	}

	if len(args) > 0 && !goja.IsUndefined(args[0]) {
		if configMap, ok := args[0].Export().(map[string]interface{}); ok {
			p.applyConfigOptions(config, configMap)
		} else if connectionString, ok := args[0].Export().(string); ok {
			config.ConnectionString = connectionString
		}
	}

	return config
}

// applyConfigOptions applies configuration options from a map.
func (p *PgModule) applyConfigOptions(config *ConnectionConfig, options map[string]interface{}) {
	if host, ok := options["host"].(string); ok {
		config.Host = host
	}
	if port, ok := options["port"].(float64); ok {
		config.Port = int(port)
	}
	if database, ok := options["database"].(string); ok {
		config.Database = database
	}
	if user, ok := options["user"].(string); ok {
		config.User = user
	}
	if password, ok := options["password"].(string); ok {
		config.Password = password
	}
	if connectionString, ok := options["connectionString"].(string); ok {
		config.ConnectionString = connectionString
	}
	if ssl, ok := options["ssl"].(string); ok {
		config.SSL = ssl
	}
	if sslmode, ok := options["sslmode"].(string); ok {
		config.SSLMode = sslmode
	}
	if timeout, ok := options["connectionTimeoutMillis"].(float64); ok {
		config.ConnectionTimeoutMs = int(timeout)
	}
	if timeout, ok := options["queryTimeoutMillis"].(float64); ok {
		config.QueryTimeoutMs = int(timeout)
	}
	if timeout, ok := options["idleTimeoutMillis"].(float64); ok {
		config.IdleTimeoutMs = int(timeout)
	}
	if max, ok := options["max"].(float64); ok {
		config.MaxConnections = int(max)
	}
	if idle, ok := options["idleCount"].(float64); ok {
		config.IdleConnections = int(idle)
	}
	if appName, ok := options["application_name"].(string); ok {
		config.ApplicationName = appName
	}
}

// buildConnectionString builds a PostgreSQL connection string from config.
func (c *ConnectionConfig) buildConnectionString() string {
	if c.ConnectionString != "" {
		return c.ConnectionString
	}

	dsn := fmt.Sprintf("host=%s port=%d dbname=%s user=%s sslmode=%s application_name=%s",
		c.Host, c.Port, c.Database, c.User, c.SSLMode, c.ApplicationName)

	if c.Password != "" {
		dsn += fmt.Sprintf(" password=%s", c.Password)
	}

	if c.ConnectionTimeoutMs > 0 {
		dsn += fmt.Sprintf(" connect_timeout=%d", c.ConnectionTimeoutMs/1000)
	}

	if c.StatementTimeoutMs > 0 {
		dsn += fmt.Sprintf(" statement_timeout=%d", c.StatementTimeoutMs)
	}

	if c.LockTimeoutMs > 0 {
		dsn += fmt.Sprintf(" lock_timeout=%d", c.LockTimeoutMs)
	}

	if c.IdleInTransactionMs > 0 {
		dsn += fmt.Sprintf(" idle_in_transaction_session_timeout=%d", c.IdleInTransactionMs)
	}

	return dsn
}

// Client methods implementation.

// connect establishes a connection to the database.
func (c *Client) connect(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("pg client connect panic: %v", r))
					}
				})
			}
		}()

		connectionString := c.config.buildConnectionString()
		db, err := sql.Open("postgres", connectionString)
		if err != nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("failed to open database connection: %w", err))
			})
			return
		}

		// Configure connection pool.
		db.SetMaxOpenConns(c.config.MaxConnections)
		db.SetMaxIdleConns(c.config.IdleConnections)
		if c.config.IdleTimeoutMs > 0 {
			db.SetConnMaxIdleTime(time.Duration(c.config.IdleTimeoutMs) * time.Millisecond)
		}

		// Test the connection.
		if err := db.Ping(); err != nil {
			_ = db.Close()
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("failed to connect to database: %w", err))
			})
			return
		}

		c.db = db

		c.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(c.runtime.ToValue(c))
		})
	}()

	return c.runtime.ToValue(promise)
}

// end closes the database connection.
func (c *Client) end(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("pg client end panic: %v", r))
					}
				})
			}
		}()

		if c.db != nil {
			err := c.db.Close()
			c.db = nil
			if err != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					_ = reject(fmt.Errorf("failed to close database connection: %w", err))
				})
				return
			}
		}

		c.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(c.runtime.ToValue(nil))
		})
	}()

	return c.runtime.ToValue(promise)
}

// query executes a SQL query.
func (c *Client) query(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("pg client query panic: %v", r))
					}
				})
			}
		}()

		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("client is not connected"))
			})
			return
		}

		if len(call.Arguments) < 1 {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("query requires at least 1 argument"))
			})
			return
		}

		queryText := call.Arguments[0].String()
		var params []interface{}

		// Parse parameters if provided.
		if len(call.Arguments) > 1 && !goja.IsUndefined(call.Arguments[1]) {
			if paramArray, ok := call.Arguments[1].Export().([]interface{}); ok {
				params = paramArray
			}
		}

		start := time.Now()
		result, err := c.executeQuery(queryText, params)
		if err != nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(err)
			})
			return
		}

		result.ProcessingMs = float64(time.Since(start).Nanoseconds()) / 1e6

		c.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(c.runtime.ToValue(result))
		})
	}()

	return c.runtime.ToValue(promise)
}

// executeQuery executes a SQL query and returns the result.
func (c *Client) executeQuery(queryText string, params []interface{}) (*QueryResult, error) {
	rows, err := c.db.Query(queryText, params...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Create fields metadata.
	fields := make([]Field, len(columns))
	for i, col := range columns {
		fields[i] = Field{
			Name:   col,
			Format: "text",
		}
	}

	// Scan all rows.
	var resultRows []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		resultRows = append(resultRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &QueryResult{
		Rows:     resultRows,
		RowCount: len(resultRows),
		Command:  "SELECT", // Simplified, would need query parsing for accurate command
		Fields:   fields,
	}, nil
}

// Placeholder methods for compatibility.

func (c *Client) release(call goja.FunctionCall) goja.Value {
	// For client, release is same as end for compatibility.
	return c.end(call)
}

func (c *Client) on(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (c *Client) off(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (c *Client) removeListener(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (c *Client) copyFrom(call goja.FunctionCall) goja.Value {
	// COPY FROM placeholder.
	promise, _, reject := c.runtime.NewPromise()
	c.loop.RunOnLoop(func(*goja.Runtime) {
		_ = reject(fmt.Errorf("copyFrom not implemented"))
	})
	return c.runtime.ToValue(promise)
}

func (c *Client) copyTo(call goja.FunctionCall) goja.Value {
	// COPY TO placeholder.
	promise, _, reject := c.runtime.NewPromise()
	c.loop.RunOnLoop(func(*goja.Runtime) {
		_ = reject(fmt.Errorf("copyTo not implemented"))
	})
	return c.runtime.ToValue(promise)
}

func (c *Client) pauseDrain(call goja.FunctionCall) goja.Value {
	// Pause drain placeholder.
	return goja.Undefined()
}

func (c *Client) resumeDrain(call goja.FunctionCall) goja.Value {
	// Resume drain placeholder.
	return goja.Undefined()
}

func (c *Client) escapeIdentifier(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return c.runtime.ToValue("")
	}
	str := call.Arguments[0].String()
	// Simple identifier escaping for PostgreSQL.
	escaped := `"` + str + `"`
	return c.runtime.ToValue(escaped)
}

func (c *Client) escapeLiteral(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return c.runtime.ToValue("")
	}
	str := call.Arguments[0].String()
	// Simple literal escaping for PostgreSQL.
	escaped := `'` + str + `'`
	return c.runtime.ToValue(escaped)
}

// Pool methods implementation (similar to Client but with pool-specific features).

func (p *Pool) connect(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := p.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("pg pool connect panic: %v", r))
					}
				})
			}
		}()

		connectionString := p.config.buildConnectionString()
		db, err := sql.Open("postgres", connectionString)
		if err != nil {
			p.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("failed to open database connection: %w", err))
			})
			return
		}

		// Configure connection pool.
		db.SetMaxOpenConns(p.config.MaxConnections)
		db.SetMaxIdleConns(p.config.IdleConnections)
		if p.config.IdleTimeoutMs > 0 {
			db.SetConnMaxIdleTime(time.Duration(p.config.IdleTimeoutMs) * time.Millisecond)
		}

		// Test the connection.
		if err := db.Ping(); err != nil {
			_ = db.Close()
			p.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("failed to connect to database: %w", err))
			})
			return
		}

		p.db = db

		// Return a client instance from the pool.
		client := &Client{
			db:      p.db,
			runtime: p.runtime,
			loop:    p.loop,
			config:  p.config,
		}

		clientObj := p.runtime.NewObject()
		p.setupClientMethodsForPoolClient(clientObj, client)

		p.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(clientObj)
		})
	}()

	return p.runtime.ToValue(promise)
}

func (p *Pool) setupClientMethodsForPoolClient(obj *goja.Object, client *Client) {
	_ = obj.Set("query", client.query)
	_ = obj.Set("release", p.releasePoolClient)
	_ = obj.Set("on", client.on)
	_ = obj.Set("off", client.off)
	_ = obj.Set("removeListener", client.removeListener)
}

func (p *Pool) releasePoolClient(call goja.FunctionCall) goja.Value {
	// Pool client release is a no-op in this implementation.
	return goja.Undefined()
}

func (p *Pool) end(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := p.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("pg pool end panic: %v", r))
					}
				})
			}
		}()

		if p.db != nil {
			err := p.db.Close()
			p.db = nil
			if err != nil {
				p.loop.RunOnLoop(func(*goja.Runtime) {
					_ = reject(fmt.Errorf("failed to close database connection: %w", err))
				})
				return
			}
		}

		p.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(p.runtime.ToValue(nil))
		})
	}()

	return p.runtime.ToValue(promise)
}

func (p *Pool) query(call goja.FunctionCall) goja.Value {
	// Pool query is same as client query.
	client := &Client{
		db:      p.db,
		runtime: p.runtime,
		loop:    p.loop,
		config:  p.config,
	}
	return client.query(call)
}

func (p *Pool) on(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (p *Pool) off(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (p *Pool) removeListener(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (p *Pool) totalCount(call goja.FunctionCall) goja.Value {
	// Return configured max connections.
	return p.runtime.ToValue(p.config.MaxConnections)
}

func (p *Pool) idleCount(call goja.FunctionCall) goja.Value {
	// Return configured idle connections.
	return p.runtime.ToValue(p.config.IdleConnections)
}

func (p *Pool) waitingCount(call goja.FunctionCall) goja.Value {
	// Return 0 for waiting connections (simplified).
	return p.runtime.ToValue(0)
}

// getNativeBindings returns native bindings information.
func (p *PgModule) getNativeBindings(call goja.FunctionCall) goja.Value {
	bindings := p.runtime.NewObject()
	_ = bindings.Set("name", "libpq")
	_ = bindings.Set("version", "1.0.0")
	return bindings
}
