package mysql2

import (
	"testing"

	"github.com/dop251/goja"
)

// MockEventLoopRunner for testing
type MockEventLoopRunner struct{}

func (m *MockEventLoopRunner) RunOnLoop(fn func(*goja.Runtime)) bool {
	vm := goja.New()
	fn(vm)
	return true
}

func TestMySQL2ModuleRegistration(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	mysql2Module := New(runtime, loop)
	err := mysql2Module.Register()
	if err != nil {
		t.Fatalf("Failed to register mysql2 module: %v", err)
	}

	// Test if mysql2 object exists
	mysql2Obj := runtime.Get("mysql2")
	if mysql2Obj == nil {
		t.Fatal("mysql2 object not found in runtime")
	}

	// Test if createConnection function exists
	mysql2ObjExported := mysql2Obj.ToObject(runtime)
	createConnection := mysql2ObjExported.Get("createConnection")
	if createConnection == nil {
		t.Fatal("createConnection function not found in mysql2 module")
	}

	// Test if createPool function exists
	createPool := mysql2ObjExported.Get("createPool")
	if createPool == nil {
		t.Fatal("createPool function not found in mysql2 module")
	}

	// Test if promise object exists
	promise := mysql2ObjExported.Get("promise")
	if promise == nil {
		t.Fatal("promise object not found in mysql2 module")
	}
}

func TestConnectionConfigParsing(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	mysql2Module := New(runtime, loop)

	// Create test arguments
	configObj := runtime.NewObject()
	_ = configObj.Set("host", "testhost")
	_ = configObj.Set("port", 3307)
	_ = configObj.Set("user", "testuser")
	_ = configObj.Set("password", "testpass")
	_ = configObj.Set("database", "testdb")
	_ = configObj.Set("charset", "utf8")
	_ = configObj.Set("connectionLimit", 20)

	args := []goja.Value{configObj}
	config := mysql2Module.parseConfig(args)

	if config.Host != "testhost" {
		t.Errorf("Expected host 'testhost', got '%s'", config.Host)
	}
	if config.Port != 3307 {
		t.Errorf("Expected port 3307, got %d", config.Port)
	}
	if config.User != "testuser" {
		t.Errorf("Expected user 'testuser', got '%s'", config.User)
	}
	if config.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", config.Password)
	}
	if config.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got '%s'", config.Database)
	}
	if config.Charset != "utf8" {
		t.Errorf("Expected charset 'utf8', got '%s'", config.Charset)
	}
	if config.ConnectionLimit != 20 {
		t.Errorf("Expected connectionLimit 20, got %d", config.ConnectionLimit)
	}
}

func TestDSNBuilding(t *testing.T) {
	config := &ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "password",
		Database: "testdb",
		Charset:  "utf8mb4",
		Timezone: "UTC",
		Timeout:  30000,
	}

	dsn := config.buildDSN()
	if dsn == "" {
		t.Fatal("DSN should not be empty")
	}

	expectedSubstrings := []string{
		"root:password@tcp(localhost:3306)/testdb",
		"charset=utf8mb4",
		"parseTime=true",
		"loc=UTC",
		"timeout=30s",
	}

	for _, substr := range expectedSubstrings {
		if !contains(dsn, substr) {
			t.Errorf("DSN should contain '%s', got: %s", substr, dsn)
		}
	}
}

func TestConnectionCreation(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	mysql2Module := New(runtime, loop)
	err := mysql2Module.Register()
	if err != nil {
		t.Fatalf("Failed to register mysql2 module: %v", err)
	}

	// Test creating a connection
	_, err = runtime.RunString(`
		var connection = mysql2.createConnection({
			host: 'localhost',
			port: 3306,
			user: 'root',
			password: 'password',
			database: 'testdb'
		});
	`)
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err)
	}

	// Verify connection has required methods
	_, err = runtime.RunString(`
		if (typeof connection.connect !== 'function') {
			throw new Error('connection.connect is not a function');
		}
		if (typeof connection.query !== 'function') {
			throw new Error('connection.query is not a function');
		}
		if (typeof connection.execute !== 'function') {
			throw new Error('connection.execute is not a function');
		}
		if (typeof connection.end !== 'function') {
			throw new Error('connection.end is not a function');
		}
	`)
	if err != nil {
		t.Fatalf("Connection missing required methods: %v", err)
	}
}

func TestPoolCreation(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	mysql2Module := New(runtime, loop)
	err := mysql2Module.Register()
	if err != nil {
		t.Fatalf("Failed to register mysql2 module: %v", err)
	}

	// Test creating a pool
	_, err = runtime.RunString(`
		var pool = mysql2.createPool({
			host: 'localhost',
			port: 3306,
			user: 'root',
			password: 'password',
			database: 'testdb',
			connectionLimit: 10
		});
	`)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Verify pool has required methods
	_, err = runtime.RunString(`
		if (typeof pool.getConnection !== 'function') {
			throw new Error('pool.getConnection is not a function');
		}
		if (typeof pool.query !== 'function') {
			throw new Error('pool.query is not a function');
		}
		if (typeof pool.execute !== 'function') {
			throw new Error('pool.execute is not a function');
		}
		if (typeof pool.end !== 'function') {
			throw new Error('pool.end is not a function');
		}
	`)
	if err != nil {
		t.Fatalf("Pool missing required methods: %v", err)
	}
}

func TestPromiseAPICreation(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	mysql2Module := New(runtime, loop)
	err := mysql2Module.Register()
	if err != nil {
		t.Fatalf("Failed to register mysql2 module: %v", err)
	}

	// Test creating a promise connection
	_, err = runtime.RunString(`
		var promiseConnection = mysql2.promise.createConnection({
			host: 'localhost',
			port: 3306,
			user: 'root',
			password: 'password',
			database: 'testdb'
		});
	`)
	if err != nil {
		t.Fatalf("Failed to create promise connection: %v", err)
	}

	// Test creating a promise pool
	_, err = runtime.RunString(`
		var promisePool = mysql2.promise.createPool({
			host: 'localhost',
			port: 3306,
			user: 'root',
			password: 'password',
			database: 'testdb',
			connectionLimit: 10
		});
	`)
	if err != nil {
		t.Fatalf("Failed to create promise pool: %v", err)
	}

	// Verify promise connection has required methods
	_, err = runtime.RunString(`
		if (typeof promiseConnection.connect !== 'function') {
			throw new Error('promiseConnection.connect is not a function');
		}
		if (typeof promiseConnection.query !== 'function') {
			throw new Error('promiseConnection.query is not a function');
		}
		if (typeof promiseConnection.execute !== 'function') {
			throw new Error('promiseConnection.execute is not a function');
		}
	`)
	if err != nil {
		t.Fatalf("Promise connection missing required methods: %v", err)
	}
}

func TestUtilityFunctions(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	mysql2Module := New(runtime, loop)
	err := mysql2Module.Register()
	if err != nil {
		t.Fatalf("Failed to register mysql2 module: %v", err)
	}

	// Test utility functions
	result, err := runtime.RunString(`
		var connection = mysql2.createConnection({});
		var escaped = connection.escape("John's data");
		var escapedId = connection.escapeId('table_name');
		var formatted = connection.format('SELECT * FROM users WHERE id = ?', [123]);
		JSON.stringify({
			escaped: escaped,
			escapedId: escapedId,
			formatted: formatted
		});
	`)
	if err != nil {
		t.Fatalf("Failed to test utility functions: %v", err)
	}

	resultStr := result.String()
	if resultStr == "" {
		t.Fatal("Utility functions should return non-empty results")
	}

	// Test module-level utility functions
	_, err = runtime.RunString(`
		var moduleEscaped = mysql2.escape("test's value");
		var moduleEscapedId = mysql2.escapeId('column_name');
		var moduleFormatted = mysql2.format('SELECT * FROM ? WHERE ? = ?', ['users', 'id', 1]);
		var moduleRaw = mysql2.raw('NOW()');
	`)
	if err != nil {
		t.Fatalf("Failed to test module-level utility functions: %v", err)
	}
}

func TestTransactionMethods(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	mysql2Module := New(runtime, loop)
	err := mysql2Module.Register()
	if err != nil {
		t.Fatalf("Failed to register mysql2 module: %v", err)
	}

	// Test transaction methods exist
	_, err = runtime.RunString(`
		var connection = mysql2.createConnection({});
		if (typeof connection.beginTransaction !== 'function') {
			throw new Error('connection.beginTransaction is not a function');
		}
		if (typeof connection.commit !== 'function') {
			throw new Error('connection.commit is not a function');
		}
		if (typeof connection.rollback !== 'function') {
			throw new Error('connection.rollback is not a function');
		}
	`)
	if err != nil {
		t.Fatalf("Connection missing transaction methods: %v", err)
	}

	// Test promise transaction methods
	_, err = runtime.RunString(`
		var promiseConnection = mysql2.promise.createConnection({});
		if (typeof promiseConnection.beginTransaction !== 'function') {
			throw new Error('promiseConnection.beginTransaction is not a function');
		}
		if (typeof promiseConnection.commit !== 'function') {
			throw new Error('promiseConnection.commit is not a function');
		}
		if (typeof promiseConnection.rollback !== 'function') {
			throw new Error('promiseConnection.rollback is not a function');
		}
	`)
	if err != nil {
		t.Fatalf("Promise connection missing transaction methods: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
