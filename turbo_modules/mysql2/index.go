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

package mysql2

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dop251/goja"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// EventLoopRunner represents the interface for running functions on the event loop.
type EventLoopRunner interface {
	RunOnLoop(fn func(*goja.Runtime)) bool
}

// MySQL2Module represents the mysql2 module for goja.
type MySQL2Module struct {
	runtime *goja.Runtime
	loop    EventLoopRunner
}

// Connection represents a MySQL connection.
type Connection struct {
	db      *sql.DB
	runtime *goja.Runtime
	loop    EventLoopRunner
	config  *ConnectionConfig
}

// Pool represents a MySQL connection pool.
type Pool struct {
	db      *sql.DB
	runtime *goja.Runtime
	loop    EventLoopRunner
	config  *ConnectionConfig
}

// PromiseConnection represents a promise-based MySQL connection.
type PromiseConnection struct {
	db      *sql.DB
	runtime *goja.Runtime
	loop    EventLoopRunner
	config  *ConnectionConfig
}

// PromisePool represents a promise-based MySQL connection pool.
type PromisePool struct {
	db      *sql.DB
	runtime *goja.Runtime
	loop    EventLoopRunner
	config  *ConnectionConfig
}

// ConnectionConfig represents the configuration for a MySQL connection.
type ConnectionConfig struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	User               string `json:"user"`
	Password           string `json:"password"`
	Database           string `json:"database"`
	Charset            string `json:"charset"`
	Timezone           string `json:"timezone"`
	Timeout            int    `json:"timeout"`
	ReadTimeout        int    `json:"readTimeout"`
	WriteTimeout       int    `json:"writeTimeout"`
	AcquireTimeout     int    `json:"acquireTimeout"`
	ConnectionLimit    int    `json:"connectionLimit"`
	QueueLimit         int    `json:"queueLimit"`
	SSL                string `json:"ssl"`
	MultipleStatements bool   `json:"multipleStatements"`
	DateStrings        bool   `json:"dateStrings"`
	Debug              bool   `json:"debug"`
	Trace              bool   `json:"trace"`
	SupportBigNumbers  bool   `json:"supportBigNumbers"`
	BigNumberStrings   bool   `json:"bigNumberStrings"`
	InsecureAuth       bool   `json:"insecureAuth"`
	TypeCast           bool   `json:"typeCast"`
	StringifyObjects   bool   `json:"stringifyObjects"`
	EnableKeepAlive    bool   `json:"enableKeepAlive"`
	KeepAliveDelay     int    `json:"keepAliveInitialDelay"`
}

// QueryResult represents the result of a query.
type QueryResult struct {
	Results      []map[string]interface{} `json:"results"`
	Fields       []Field                  `json:"fields"`
	AffectedRows int64                    `json:"affectedRows"`
	InsertId     int64                    `json:"insertId"`
	ChangedRows  int64                    `json:"changedRows"`
	ServerStatus int                      `json:"serverStatus"`
	WarningCount int                      `json:"warningCount"`
	Message      string                   `json:"message"`
}

// Field represents a database field metadata.
type Field struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Length     int    `json:"length"`
	Decimals   int    `json:"decimals"`
	Flags      int    `json:"flags"`
	Default    string `json:"default"`
	ZeroFill   bool   `json:"zerofill"`
	Protocol41 bool   `json:"protocol41"`
	CharsetNr  int    `json:"charsetNr"`
	Database   string `json:"database"`
	Table      string `json:"table"`
	OrgTable   string `json:"orgTable"`
	OrgName    string `json:"orgName"`
}

// New creates a new MySQL2 module instance.
func New(runtime *goja.Runtime, loop EventLoopRunner) *MySQL2Module {
	return &MySQL2Module{
		runtime: runtime,
		loop:    loop,
	}
}

// Register registers the mysql2 module with the goja runtime.
func (m *MySQL2Module) Register() error {
	module := m.runtime.NewObject()

	// Create connection function
	if err := module.Set("createConnection", m.createConnection); err != nil {
		return fmt.Errorf("failed to set createConnection: %w", err)
	}

	// Create pool function
	if err := module.Set("createPool", m.createPool); err != nil {
		return fmt.Errorf("failed to set createPool: %w", err)
	}

	// Create promise connection function
	if err := module.Set("createConnectionPromise", m.createConnectionPromise); err != nil {
		return fmt.Errorf("failed to set createConnectionPromise: %w", err)
	}

	// Create promise pool function
	if err := module.Set("createPoolPromise", m.createPoolPromise); err != nil {
		return fmt.Errorf("failed to set createPoolPromise: %w", err)
	}

	// Format function for building queries
	if err := module.Set("format", m.format); err != nil {
		return fmt.Errorf("failed to set format function: %w", err)
	}

	// Escape function for escaping values
	if err := module.Set("escape", m.escape); err != nil {
		return fmt.Errorf("failed to set escape function: %w", err)
	}

	// Escape ID function for escaping identifiers
	if err := module.Set("escapeId", m.escapeId); err != nil {
		return fmt.Errorf("failed to set escapeId function: %w", err)
	}

	// Raw function for raw values
	if err := module.Set("raw", m.raw); err != nil {
		return fmt.Errorf("failed to set raw function: %w", err)
	}

	// Default export for promise-based API
	promise := m.runtime.NewObject()
	_ = promise.Set("createConnection", m.createConnectionPromise)
	_ = promise.Set("createPool", m.createPoolPromise)
	_ = promise.Set("format", m.format)
	_ = promise.Set("escape", m.escape)
	_ = promise.Set("escapeId", m.escapeId)
	_ = promise.Set("raw", m.raw)

	if err := module.Set("promise", promise); err != nil {
		return fmt.Errorf("failed to set promise export: %w", err)
	}

	return m.runtime.Set("mysql2", module)
}

// parseConfig parses the configuration from function call arguments.
func (m *MySQL2Module) parseConfig(args []goja.Value) *ConnectionConfig {
	config := &ConnectionConfig{
		Host:               "localhost",
		Port:               3306,
		User:               "root",
		Password:           "",
		Database:           "",
		Charset:            "utf8mb4",
		Timezone:           "local",
		Timeout:            60000,
		ReadTimeout:        30000,
		WriteTimeout:       30000,
		AcquireTimeout:     60000,
		ConnectionLimit:    10,
		QueueLimit:         0,
		SSL:                "false",
		MultipleStatements: false,
		DateStrings:        false,
		Debug:              false,
		Trace:              false,
		SupportBigNumbers:  false,
		BigNumberStrings:   false,
		InsecureAuth:       false,
		TypeCast:           true,
		StringifyObjects:   false,
		EnableKeepAlive:    true,
		KeepAliveDelay:     0,
	}

	if len(args) > 0 && !goja.IsUndefined(args[0]) {
		if configMap, ok := args[0].Export().(map[string]interface{}); ok {
			m.applyConfigOptions(config, configMap)
		}
	}

	return config
}

// parseNumeric parses a numeric value that can be int, int64, float64, etc.
func (m *MySQL2Module) parseNumeric(value interface{}) *float64 {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case int32:
		f := float64(v)
		return &f
	default:
		return nil
	}
}

// applyConfigOptions applies configuration options from a map.
func (m *MySQL2Module) applyConfigOptions(config *ConnectionConfig, options map[string]interface{}) {
	if host, ok := options["host"].(string); ok {
		config.Host = host
	}
	if port := m.parseNumeric(options["port"]); port != nil {
		config.Port = int(*port)
	}
	if user, ok := options["user"].(string); ok {
		config.User = user
	}
	if password, ok := options["password"].(string); ok {
		config.Password = password
	}
	if database, ok := options["database"].(string); ok {
		config.Database = database
	}
	if charset, ok := options["charset"].(string); ok {
		config.Charset = charset
	}
	if timezone, ok := options["timezone"].(string); ok {
		config.Timezone = timezone
	}
	if timeout := m.parseNumeric(options["timeout"]); timeout != nil {
		config.Timeout = int(*timeout)
	}
	if readTimeout := m.parseNumeric(options["readTimeout"]); readTimeout != nil {
		config.ReadTimeout = int(*readTimeout)
	}
	if writeTimeout := m.parseNumeric(options["writeTimeout"]); writeTimeout != nil {
		config.WriteTimeout = int(*writeTimeout)
	}
	if acquireTimeout := m.parseNumeric(options["acquireTimeout"]); acquireTimeout != nil {
		config.AcquireTimeout = int(*acquireTimeout)
	}
	if connectionLimit := m.parseNumeric(options["connectionLimit"]); connectionLimit != nil {
		config.ConnectionLimit = int(*connectionLimit)
	}
	if queueLimit := m.parseNumeric(options["queueLimit"]); queueLimit != nil {
		config.QueueLimit = int(*queueLimit)
	}
	if ssl, ok := options["ssl"].(string); ok {
		config.SSL = ssl
	}
	if multipleStatements, ok := options["multipleStatements"].(bool); ok {
		config.MultipleStatements = multipleStatements
	}
	if dateStrings, ok := options["dateStrings"].(bool); ok {
		config.DateStrings = dateStrings
	}
	if debug, ok := options["debug"].(bool); ok {
		config.Debug = debug
	}
	if trace, ok := options["trace"].(bool); ok {
		config.Trace = trace
	}
	if supportBigNumbers, ok := options["supportBigNumbers"].(bool); ok {
		config.SupportBigNumbers = supportBigNumbers
	}
	if bigNumberStrings, ok := options["bigNumberStrings"].(bool); ok {
		config.BigNumberStrings = bigNumberStrings
	}
	if insecureAuth, ok := options["insecureAuth"].(bool); ok {
		config.InsecureAuth = insecureAuth
	}
	if typeCast, ok := options["typeCast"].(bool); ok {
		config.TypeCast = typeCast
	}
	if stringifyObjects, ok := options["stringifyObjects"].(bool); ok {
		config.StringifyObjects = stringifyObjects
	}
	if enableKeepAlive, ok := options["enableKeepAlive"].(bool); ok {
		config.EnableKeepAlive = enableKeepAlive
	}
	if keepAliveDelay := m.parseNumeric(options["keepAliveInitialDelay"]); keepAliveDelay != nil {
		config.KeepAliveDelay = int(*keepAliveDelay)
	}
}

// buildDSN builds a MySQL DSN from config.
func (c *ConnectionConfig) buildDSN() string {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Charset, c.Timezone)

	if c.Timeout > 0 {
		dsn += fmt.Sprintf("&timeout=%ds", c.Timeout/1000)
	}
	if c.ReadTimeout > 0 {
		dsn += fmt.Sprintf("&readTimeout=%ds", c.ReadTimeout/1000)
	}
	if c.WriteTimeout > 0 {
		dsn += fmt.Sprintf("&writeTimeout=%ds", c.WriteTimeout/1000)
	}
	if c.MultipleStatements {
		dsn += "&multiStatements=true"
	}
	if c.SSL != "false" {
		dsn += "&tls=" + c.SSL
	}

	return dsn
}

// createConnection creates a callback-based MySQL connection.
func (m *MySQL2Module) createConnection(call goja.FunctionCall) goja.Value {
	config := m.parseConfig(call.Arguments)
	connection := &Connection{
		runtime: m.runtime,
		loop:    m.loop,
		config:  config,
	}

	obj := m.runtime.NewObject()
	m.setupConnectionMethods(obj, connection)
	return obj
}

// createPool creates a callback-based MySQL connection pool.
func (m *MySQL2Module) createPool(call goja.FunctionCall) goja.Value {
	config := m.parseConfig(call.Arguments)
	pool := &Pool{
		runtime: m.runtime,
		loop:    m.loop,
		config:  config,
	}

	obj := m.runtime.NewObject()
	m.setupPoolMethods(obj, pool)
	return obj
}

// createConnectionPromise creates a promise-based MySQL connection.
func (m *MySQL2Module) createConnectionPromise(call goja.FunctionCall) goja.Value {
	config := m.parseConfig(call.Arguments)
	connection := &PromiseConnection{
		runtime: m.runtime,
		loop:    m.loop,
		config:  config,
	}

	obj := m.runtime.NewObject()
	m.setupPromiseConnectionMethods(obj, connection)
	return obj
}

// createPoolPromise creates a promise-based MySQL connection pool.
func (m *MySQL2Module) createPoolPromise(call goja.FunctionCall) goja.Value {
	config := m.parseConfig(call.Arguments)
	pool := &PromisePool{
		runtime: m.runtime,
		loop:    m.loop,
		config:  config,
	}

	obj := m.runtime.NewObject()
	m.setupPromisePoolMethods(obj, pool)
	return obj
}

// setupConnectionMethods sets up callback-based connection methods.
func (m *MySQL2Module) setupConnectionMethods(obj *goja.Object, conn *Connection) {
	_ = obj.Set("connect", conn.connect)
	_ = obj.Set("end", conn.end)
	_ = obj.Set("destroy", conn.destroy)
	_ = obj.Set("query", conn.query)
	_ = obj.Set("execute", conn.execute)
	_ = obj.Set("beginTransaction", conn.beginTransaction)
	_ = obj.Set("commit", conn.commit)
	_ = obj.Set("rollback", conn.rollback)
	_ = obj.Set("changeUser", conn.changeUser)
	_ = obj.Set("ping", conn.ping)
	_ = obj.Set("statistics", conn.statistics)
	_ = obj.Set("format", conn.format)
	_ = obj.Set("escape", conn.escape)
	_ = obj.Set("escapeId", conn.escapeId)
	_ = obj.Set("on", conn.on)
	_ = obj.Set("off", conn.off)
	_ = obj.Set("removeListener", conn.removeListener)
	_ = obj.Set("pause", conn.pause)
	_ = obj.Set("resume", conn.resume)
}

// setupPoolMethods sets up callback-based pool methods.
func (m *MySQL2Module) setupPoolMethods(obj *goja.Object, pool *Pool) {
	_ = obj.Set("getConnection", pool.getConnection)
	_ = obj.Set("releaseConnection", pool.releaseConnection)
	_ = obj.Set("end", pool.end)
	_ = obj.Set("query", pool.query)
	_ = obj.Set("execute", pool.execute)
	_ = obj.Set("on", pool.on)
	_ = obj.Set("off", pool.off)
	_ = obj.Set("removeListener", pool.removeListener)
}

// setupPromiseConnectionMethods sets up promise-based connection methods.
func (m *MySQL2Module) setupPromiseConnectionMethods(obj *goja.Object, conn *PromiseConnection) {
	_ = obj.Set("connect", conn.connect)
	_ = obj.Set("end", conn.end)
	_ = obj.Set("destroy", conn.destroy)
	_ = obj.Set("query", conn.query)
	_ = obj.Set("execute", conn.execute)
	_ = obj.Set("beginTransaction", conn.beginTransaction)
	_ = obj.Set("commit", conn.commit)
	_ = obj.Set("rollback", conn.rollback)
	_ = obj.Set("changeUser", conn.changeUser)
	_ = obj.Set("ping", conn.ping)
	_ = obj.Set("statistics", conn.statistics)
	_ = obj.Set("format", conn.format)
	_ = obj.Set("escape", conn.escape)
	_ = obj.Set("escapeId", conn.escapeId)
}

// setupPromisePoolMethods sets up promise-based pool methods.
func (m *MySQL2Module) setupPromisePoolMethods(obj *goja.Object, pool *PromisePool) {
	_ = obj.Set("getConnection", pool.getConnection)
	_ = obj.Set("end", pool.end)
	_ = obj.Set("query", pool.query)
	_ = obj.Set("execute", pool.execute)
}

// Connection methods implementation (callback-based).

func (c *Connection) connect(call goja.FunctionCall) goja.Value {
	callback := call.Arguments[0]

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if fn, ok := goja.AssertFunction(callback); ok {
						err := fmt.Errorf("mysql2 connection connect panic: %v", r)
						_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
					}
				})
			}
		}()

		dsn := c.config.buildDSN()
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if fn, ok := goja.AssertFunction(callback); ok {
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				}
			})
			return
		}

		// Configure connection pool.
		db.SetMaxOpenConns(c.config.ConnectionLimit)
		db.SetMaxIdleConns(c.config.ConnectionLimit / 2)
		if c.config.Timeout > 0 {
			db.SetConnMaxLifetime(time.Duration(c.config.Timeout) * time.Millisecond)
		}

		// Test the connection.
		if err := db.Ping(); err != nil {
			_ = db.Close()
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if fn, ok := goja.AssertFunction(callback); ok {
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				}
			})
			return
		}

		c.db = db

		c.loop.RunOnLoop(func(*goja.Runtime) {
			if fn, ok := goja.AssertFunction(callback); ok {
				_, _ = fn(goja.Undefined(), goja.Null())
			}
		})
	}()

	return goja.Undefined()
}

func (c *Connection) end(call goja.FunctionCall) goja.Value {
	var callback goja.Value
	if len(call.Arguments) > 0 {
		callback = call.Arguments[0]
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if !goja.IsUndefined(callback) {
						if fn, ok := goja.AssertFunction(callback); ok {
							err := fmt.Errorf("mysql2 connection end panic: %v", r)
							_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
						}
					}
				})
			}
		}()

		if c.db != nil {
			err := c.db.Close()
			c.db = nil

			c.loop.RunOnLoop(func(*goja.Runtime) {
				if !goja.IsUndefined(callback) {
					if fn, ok := goja.AssertFunction(callback); ok {
						if err != nil {
							_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
						} else {
							_, _ = fn(goja.Undefined(), goja.Null())
						}
					}
				}
			})
		} else {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if !goja.IsUndefined(callback) {
					if fn, ok := goja.AssertFunction(callback); ok {
						_, _ = fn(goja.Undefined(), goja.Null())
					}
				}
			})
		}
	}()

	return goja.Undefined()
}

func (c *Connection) query(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(c.runtime.NewTypeError("query requires at least 1 argument"))
	}

	queryText := call.Arguments[0].String()
	var params []interface{}
	var callback goja.Value

	// Parse arguments (sql, [values], callback)
	if len(call.Arguments) == 2 {
		callback = call.Arguments[1]
	} else if len(call.Arguments) >= 3 {
		if paramArray, ok := call.Arguments[1].Export().([]interface{}); ok {
			params = paramArray
		}
		callback = call.Arguments[2]
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if fn, ok := goja.AssertFunction(callback); ok {
						err := fmt.Errorf("mysql2 connection query panic: %v", r)
						_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
					}
				})
			}
		}()

		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if fn, ok := goja.AssertFunction(callback); ok {
					err := fmt.Errorf("connection is not established")
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				}
			})
			return
		}

		result, err := c.executeQuery(queryText, params)
		if err != nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if fn, ok := goja.AssertFunction(callback); ok {
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				}
			})
			return
		}

		c.loop.RunOnLoop(func(*goja.Runtime) {
			if fn, ok := goja.AssertFunction(callback); ok {
				_, _ = fn(goja.Undefined(), goja.Null(), c.runtime.ToValue(result.Results), c.runtime.ToValue(result.Fields))
			}
		})
	}()

	return goja.Undefined()
}

// executeQuery executes a SQL query and returns the result.
func (c *Connection) executeQuery(queryText string, params []interface{}) (*QueryResult, error) {
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
			Type:   "VARCHAR", // Simplified type
			Length: 255,
		}
	}

	// Scan all rows.
	var results []map[string]interface{}
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
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return &QueryResult{
		Results: results,
		Fields:  fields,
	}, nil
}

// Placeholder methods for Connection.

func (c *Connection) destroy(call goja.FunctionCall) goja.Value {
	// Immediate connection close without callback.
	if c.db != nil {
		_ = c.db.Close()
		c.db = nil
	}
	return goja.Undefined()
}

func (c *Connection) execute(call goja.FunctionCall) goja.Value {
	// Execute is same as query for this implementation.
	return c.query(call)
}

func (c *Connection) beginTransaction(call goja.FunctionCall) goja.Value {
	callback := call.Arguments[0]
	go func() {
		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if fn, ok := goja.AssertFunction(callback); ok {
					err := fmt.Errorf("connection is not established")
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				}
			})
			return
		}

		_, err := c.db.Exec("START TRANSACTION")
		c.loop.RunOnLoop(func(*goja.Runtime) {
			if fn, ok := goja.AssertFunction(callback); ok {
				if err != nil {
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				} else {
					_, _ = fn(goja.Undefined(), goja.Null())
				}
			}
		})
	}()
	return goja.Undefined()
}

func (c *Connection) commit(call goja.FunctionCall) goja.Value {
	callback := call.Arguments[0]
	go func() {
		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if fn, ok := goja.AssertFunction(callback); ok {
					err := fmt.Errorf("connection is not established")
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				}
			})
			return
		}

		_, err := c.db.Exec("COMMIT")
		c.loop.RunOnLoop(func(*goja.Runtime) {
			if fn, ok := goja.AssertFunction(callback); ok {
				if err != nil {
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				} else {
					_, _ = fn(goja.Undefined(), goja.Null())
				}
			}
		})
	}()
	return goja.Undefined()
}

func (c *Connection) rollback(call goja.FunctionCall) goja.Value {
	callback := call.Arguments[0]
	go func() {
		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if fn, ok := goja.AssertFunction(callback); ok {
					err := fmt.Errorf("connection is not established")
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				}
			})
			return
		}

		_, err := c.db.Exec("ROLLBACK")
		c.loop.RunOnLoop(func(*goja.Runtime) {
			if fn, ok := goja.AssertFunction(callback); ok {
				if err != nil {
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				} else {
					_, _ = fn(goja.Undefined(), goja.Null())
				}
			}
		})
	}()
	return goja.Undefined()
}

func (c *Connection) changeUser(call goja.FunctionCall) goja.Value {
	// Placeholder implementation.
	callback := call.Arguments[1]
	c.loop.RunOnLoop(func(*goja.Runtime) {
		if fn, ok := goja.AssertFunction(callback); ok {
			err := fmt.Errorf("changeUser not implemented")
			_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
		}
	})
	return goja.Undefined()
}

func (c *Connection) ping(call goja.FunctionCall) goja.Value {
	callback := call.Arguments[0]
	go func() {
		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				if fn, ok := goja.AssertFunction(callback); ok {
					err := fmt.Errorf("connection is not established")
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				}
			})
			return
		}

		err := c.db.Ping()
		c.loop.RunOnLoop(func(*goja.Runtime) {
			if fn, ok := goja.AssertFunction(callback); ok {
				if err != nil {
					_, _ = fn(goja.Undefined(), c.runtime.ToValue(err))
				} else {
					_, _ = fn(goja.Undefined(), goja.Null())
				}
			}
		})
	}()
	return goja.Undefined()
}

func (c *Connection) statistics(call goja.FunctionCall) goja.Value {
	callback := call.Arguments[0]
	c.loop.RunOnLoop(func(*goja.Runtime) {
		if fn, ok := goja.AssertFunction(callback); ok {
			stats := map[string]interface{}{
				"Uptime": 0,
			}
			_, _ = fn(goja.Undefined(), goja.Null(), c.runtime.ToValue(stats))
		}
	})
	return goja.Undefined()
}

func (c *Connection) format(call goja.FunctionCall) goja.Value {
	// Simple format implementation - just return the query.
	if len(call.Arguments) > 0 {
		return call.Arguments[0]
	}
	return c.runtime.ToValue("")
}

func (c *Connection) escape(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return c.runtime.ToValue("")
	}
	str := call.Arguments[0].String()
	// Simple escaping - replace single quotes.
	escaped := "'" + str + "'"
	return c.runtime.ToValue(escaped)
}

func (c *Connection) escapeId(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return c.runtime.ToValue("")
	}
	str := call.Arguments[0].String()
	// Simple identifier escaping for MySQL.
	escaped := "`" + str + "`"
	return c.runtime.ToValue(escaped)
}

func (c *Connection) on(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (c *Connection) off(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (c *Connection) removeListener(call goja.FunctionCall) goja.Value {
	// Event handling placeholder.
	return goja.Undefined()
}

func (c *Connection) pause(call goja.FunctionCall) goja.Value {
	// Pause placeholder.
	return goja.Undefined()
}

func (c *Connection) resume(call goja.FunctionCall) goja.Value {
	// Resume placeholder.
	return goja.Undefined()
}

// Pool methods implementation (callback-based).

func (p *Pool) getConnection(call goja.FunctionCall) goja.Value {
	callback := call.Arguments[0]

	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.loop.RunOnLoop(func(*goja.Runtime) {
					if fn, ok := goja.AssertFunction(callback); ok {
						err := fmt.Errorf("mysql2 pool getConnection panic: %v", r)
						_, _ = fn(goja.Undefined(), p.runtime.ToValue(err))
					}
				})
			}
		}()

		if p.db == nil {
			dsn := p.config.buildDSN()
			db, err := sql.Open("mysql", dsn)
			if err != nil {
				p.loop.RunOnLoop(func(*goja.Runtime) {
					if fn, ok := goja.AssertFunction(callback); ok {
						_, _ = fn(goja.Undefined(), p.runtime.ToValue(err))
					}
				})
				return
			}

			// Configure connection pool.
			db.SetMaxOpenConns(p.config.ConnectionLimit)
			db.SetMaxIdleConns(p.config.ConnectionLimit / 2)

			// Test the connection.
			if err := db.Ping(); err != nil {
				_ = db.Close()
				p.loop.RunOnLoop(func(*goja.Runtime) {
					if fn, ok := goja.AssertFunction(callback); ok {
						_, _ = fn(goja.Undefined(), p.runtime.ToValue(err))
					}
				})
				return
			}

			p.db = db
		}

		// Return a connection-like object.
		conn := &Connection{
			db:      p.db,
			runtime: p.runtime,
			loop:    p.loop,
			config:  p.config,
		}

		connObj := p.runtime.NewObject()
		p.setupPoolConnectionMethods(connObj, conn)

		p.loop.RunOnLoop(func(*goja.Runtime) {
			if fn, ok := goja.AssertFunction(callback); ok {
				_, _ = fn(goja.Undefined(), goja.Null(), connObj)
			}
		})
	}()

	return goja.Undefined()
}

func (p *Pool) setupPoolConnectionMethods(obj *goja.Object, conn *Connection) {
	_ = obj.Set("query", conn.query)
	_ = obj.Set("execute", conn.execute)
	_ = obj.Set("beginTransaction", conn.beginTransaction)
	_ = obj.Set("commit", conn.commit)
	_ = obj.Set("rollback", conn.rollback)
	_ = obj.Set("ping", conn.ping)
	_ = obj.Set("format", conn.format)
	_ = obj.Set("escape", conn.escape)
	_ = obj.Set("escapeId", conn.escapeId)
	_ = obj.Set("release", p.releaseConnection)
	_ = obj.Set("destroy", conn.destroy)
}

func (p *Pool) releaseConnection(call goja.FunctionCall) goja.Value {
	// Pool connection release is a no-op in this implementation.
	return goja.Undefined()
}

func (p *Pool) end(call goja.FunctionCall) goja.Value {
	var callback goja.Value
	if len(call.Arguments) > 0 {
		callback = call.Arguments[0]
	}

	go func() {
		if p.db != nil {
			err := p.db.Close()
			p.db = nil

			p.loop.RunOnLoop(func(*goja.Runtime) {
				if !goja.IsUndefined(callback) {
					if fn, ok := goja.AssertFunction(callback); ok {
						if err != nil {
							_, _ = fn(goja.Undefined(), p.runtime.ToValue(err))
						} else {
							_, _ = fn(goja.Undefined(), goja.Null())
						}
					}
				}
			})
		}
	}()

	return goja.Undefined()
}

func (p *Pool) query(call goja.FunctionCall) goja.Value {
	// Pool query is same as connection query.
	conn := &Connection{
		db:      p.db,
		runtime: p.runtime,
		loop:    p.loop,
		config:  p.config,
	}
	return conn.query(call)
}

func (p *Pool) execute(call goja.FunctionCall) goja.Value {
	// Pool execute is same as connection execute.
	conn := &Connection{
		db:      p.db,
		runtime: p.runtime,
		loop:    p.loop,
		config:  p.config,
	}
	return conn.execute(call)
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

// Promise-based implementations.

func (c *PromiseConnection) connect(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("mysql2 promise connection connect panic: %v", r))
					}
				})
			}
		}()

		dsn := c.config.buildDSN()
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("failed to open database connection: %w", err))
			})
			return
		}

		// Configure connection pool.
		db.SetMaxOpenConns(c.config.ConnectionLimit)
		db.SetMaxIdleConns(c.config.ConnectionLimit / 2)

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

func (c *PromiseConnection) end(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("mysql2 promise connection end panic: %v", r))
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

func (c *PromiseConnection) query(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("mysql2 promise connection query panic: %v", r))
					}
				})
			}
		}()

		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("connection is not established"))
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

		conn := &Connection{
			db:      c.db,
			runtime: c.runtime,
			loop:    c.loop,
			config:  c.config,
		}

		result, err := conn.executeQuery(queryText, params)
		if err != nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(err)
			})
			return
		}

		c.loop.RunOnLoop(func(*goja.Runtime) {
			resultArray := c.runtime.NewArray()
			_ = resultArray.Set("0", c.runtime.ToValue(result.Results))
			_ = resultArray.Set("1", c.runtime.ToValue(result.Fields))
			resolve(resultArray)
		})
	}()

	return c.runtime.ToValue(promise)
}

// Placeholder promise methods.

func (c *PromiseConnection) destroy(call goja.FunctionCall) goja.Value {
	if c.db != nil {
		_ = c.db.Close()
		c.db = nil
	}
	return goja.Undefined()
}

func (c *PromiseConnection) execute(call goja.FunctionCall) goja.Value {
	return c.query(call)
}

func (c *PromiseConnection) beginTransaction(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()
	go func() {
		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("connection is not established"))
			})
			return
		}
		_, err := c.db.Exec("START TRANSACTION")
		c.loop.RunOnLoop(func(*goja.Runtime) {
			if err != nil {
				_ = reject(err)
			} else {
				_ = resolve(c.runtime.ToValue(nil))
			}
		})
	}()
	return c.runtime.ToValue(promise)
}

func (c *PromiseConnection) commit(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()
	go func() {
		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("connection is not established"))
			})
			return
		}
		_, err := c.db.Exec("COMMIT")
		c.loop.RunOnLoop(func(*goja.Runtime) {
			if err != nil {
				_ = reject(err)
			} else {
				_ = resolve(c.runtime.ToValue(nil))
			}
		})
	}()
	return c.runtime.ToValue(promise)
}

func (c *PromiseConnection) rollback(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()
	go func() {
		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("connection is not established"))
			})
			return
		}
		_, err := c.db.Exec("ROLLBACK")
		c.loop.RunOnLoop(func(*goja.Runtime) {
			if err != nil {
				_ = reject(err)
			} else {
				_ = resolve(c.runtime.ToValue(nil))
			}
		})
	}()
	return c.runtime.ToValue(promise)
}

func (c *PromiseConnection) changeUser(call goja.FunctionCall) goja.Value {
	promise, _, reject := c.runtime.NewPromise()
	c.loop.RunOnLoop(func(*goja.Runtime) {
		_ = reject(fmt.Errorf("changeUser not implemented"))
	})
	return c.runtime.ToValue(promise)
}

func (c *PromiseConnection) ping(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := c.runtime.NewPromise()
	go func() {
		if c.db == nil {
			c.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("connection is not established"))
			})
			return
		}
		err := c.db.Ping()
		c.loop.RunOnLoop(func(*goja.Runtime) {
			if err != nil {
				_ = reject(err)
			} else {
				_ = resolve(c.runtime.ToValue(nil))
			}
		})
	}()
	return c.runtime.ToValue(promise)
}

func (c *PromiseConnection) statistics(call goja.FunctionCall) goja.Value {
	promise, resolve, _ := c.runtime.NewPromise()
	c.loop.RunOnLoop(func(*goja.Runtime) {
		stats := map[string]interface{}{
			"Uptime": 0,
		}
		_ = resolve(c.runtime.ToValue(stats))
	})
	return c.runtime.ToValue(promise)
}

func (c *PromiseConnection) format(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) > 0 {
		return call.Arguments[0]
	}
	return c.runtime.ToValue("")
}

func (c *PromiseConnection) escape(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return c.runtime.ToValue("")
	}
	str := call.Arguments[0].String()
	escaped := "'" + str + "'"
	return c.runtime.ToValue(escaped)
}

func (c *PromiseConnection) escapeId(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return c.runtime.ToValue("")
	}
	str := call.Arguments[0].String()
	escaped := "`" + str + "`"
	return c.runtime.ToValue(escaped)
}

// Promise pool methods.

func (p *PromisePool) getConnection(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := p.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("mysql2 promise pool getConnection panic: %v", r))
					}
				})
			}
		}()

		if p.db == nil {
			dsn := p.config.buildDSN()
			db, err := sql.Open("mysql", dsn)
			if err != nil {
				p.loop.RunOnLoop(func(*goja.Runtime) {
					_ = reject(fmt.Errorf("failed to open database connection: %w", err))
				})
				return
			}

			// Configure connection pool.
			db.SetMaxOpenConns(p.config.ConnectionLimit)
			db.SetMaxIdleConns(p.config.ConnectionLimit / 2)

			// Test the connection.
			if err := db.Ping(); err != nil {
				_ = db.Close()
				p.loop.RunOnLoop(func(*goja.Runtime) {
					_ = reject(fmt.Errorf("failed to connect to database: %w", err))
				})
				return
			}

			p.db = db
		}

		// Return a promise connection.
		conn := &PromiseConnection{
			db:      p.db,
			runtime: p.runtime,
			loop:    p.loop,
			config:  p.config,
		}

		connObj := p.runtime.NewObject()
		p.setupPromisePoolConnectionMethods(connObj, conn)

		p.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(connObj)
		})
	}()

	return p.runtime.ToValue(promise)
}

func (p *PromisePool) setupPromisePoolConnectionMethods(obj *goja.Object, conn *PromiseConnection) {
	_ = obj.Set("query", conn.query)
	_ = obj.Set("execute", conn.execute)
	_ = obj.Set("beginTransaction", conn.beginTransaction)
	_ = obj.Set("commit", conn.commit)
	_ = obj.Set("rollback", conn.rollback)
	_ = obj.Set("ping", conn.ping)
	_ = obj.Set("format", conn.format)
	_ = obj.Set("escape", conn.escape)
	_ = obj.Set("escapeId", conn.escapeId)
	_ = obj.Set("release", p.releasePromiseConnection)
	_ = obj.Set("destroy", conn.destroy)
}

func (p *PromisePool) releasePromiseConnection(call goja.FunctionCall) goja.Value {
	// Pool connection release is a no-op.
	return goja.Undefined()
}

func (p *PromisePool) end(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := p.runtime.NewPromise()

	go func() {
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

func (p *PromisePool) query(call goja.FunctionCall) goja.Value {
	conn := &PromiseConnection{
		db:      p.db,
		runtime: p.runtime,
		loop:    p.loop,
		config:  p.config,
	}
	return conn.query(call)
}

func (p *PromisePool) execute(call goja.FunctionCall) goja.Value {
	conn := &PromiseConnection{
		db:      p.db,
		runtime: p.runtime,
		loop:    p.loop,
		config:  p.config,
	}
	return conn.execute(call)
}

// Utility functions.

func (m *MySQL2Module) format(call goja.FunctionCall) goja.Value {
	// Simple format implementation.
	if len(call.Arguments) > 0 {
		return call.Arguments[0]
	}
	return m.runtime.ToValue("")
}

func (m *MySQL2Module) escape(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return m.runtime.ToValue("")
	}
	str := call.Arguments[0].String()
	escaped := "'" + str + "'"
	return m.runtime.ToValue(escaped)
}

func (m *MySQL2Module) escapeId(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		return m.runtime.ToValue("")
	}
	str := call.Arguments[0].String()
	escaped := "`" + str + "`"
	return m.runtime.ToValue(escaped)
}

func (m *MySQL2Module) raw(call goja.FunctionCall) goja.Value {
	// Raw function returns the value as-is.
	if len(call.Arguments) > 0 {
		return call.Arguments[0]
	}
	return m.runtime.ToValue("")
}
