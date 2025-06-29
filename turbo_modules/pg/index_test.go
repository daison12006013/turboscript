package pg

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

func TestPgModuleRegistration(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	pgModule := New(runtime, loop)
	err := pgModule.Register()
	if err != nil {
		t.Fatalf("Failed to register pg module: %v", err)
	}

	// Test if pg object exists
	pgObj := runtime.Get("pg")
	if pgObj == nil {
		t.Fatal("pg object not found in runtime")
	}

	// Test if Client constructor exists
	pgObjExported := pgObj.ToObject(runtime)
	clientConstructor := pgObjExported.Get("Client")
	if clientConstructor == nil {
		t.Fatal("Client constructor not found in pg module")
	}

	// Test if Pool constructor exists
	poolConstructor := pgObjExported.Get("Pool")
	if poolConstructor == nil {
		t.Fatal("Pool constructor not found in pg module")
	}

	// Test if defaults exist
	defaults := pgObjExported.Get("defaults")
	if defaults == nil {
		t.Fatal("defaults not found in pg module")
	}
}

func TestConnectionConfig(t *testing.T) {
	config := &ConnectionConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "test",
		User:     "postgres",
		Password: "password",
		SSLMode:  "prefer",
	}

	dsn := config.buildConnectionString()
	if dsn == "" {
		t.Fatal("DSN should not be empty")
	}

	expectedSubstrings := []string{
		"host=localhost",
		"port=5432",
		"dbname=test",
		"user=postgres",
		"sslmode=prefer",
	}

	for _, substr := range expectedSubstrings {
		if !contains(dsn, substr) {
			t.Errorf("DSN should contain '%s', got: %s", substr, dsn)
		}
	}
}

func TestClientCreation(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	pgModule := New(runtime, loop)
	err := pgModule.Register()
	if err != nil {
		t.Fatalf("Failed to register pg module: %v", err)
	}

	// Test creating a client
	_, err = runtime.RunString(`
		var client = new pg.Client({
			host: 'localhost',
			port: 5432,
			database: 'test',
			user: 'postgres',
			password: 'password'
		});
	`)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Verify client has required methods
	_, err = runtime.RunString(`
		if (typeof client.connect !== 'function') {
			throw new Error('client.connect is not a function');
		}
		if (typeof client.query !== 'function') {
			throw new Error('client.query is not a function');
		}
		if (typeof client.end !== 'function') {
			throw new Error('client.end is not a function');
		}
	`)
	if err != nil {
		t.Fatalf("Client missing required methods: %v", err)
	}
}

func TestPoolCreation(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	pgModule := New(runtime, loop)
	err := pgModule.Register()
	if err != nil {
		t.Fatalf("Failed to register pg module: %v", err)
	}

	// Test creating a pool
	_, err = runtime.RunString(`
		var pool = new pg.Pool({
			host: 'localhost',
			port: 5432,
			database: 'test',
			user: 'postgres',
			password: 'password',
			max: 10
		});
	`)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Verify pool has required methods
	_, err = runtime.RunString(`
		if (typeof pool.connect !== 'function') {
			throw new Error('pool.connect is not a function');
		}
		if (typeof pool.query !== 'function') {
			throw new Error('pool.query is not a function');
		}
		if (typeof pool.end !== 'function') {
			throw new Error('pool.end is not a function');
		}
	`)
	if err != nil {
		t.Fatalf("Pool missing required methods: %v", err)
	}
}

func TestEscapeFunctions(t *testing.T) {
	runtime := goja.New()
	loop := &MockEventLoopRunner{}

	pgModule := New(runtime, loop)
	err := pgModule.Register()
	if err != nil {
		t.Fatalf("Failed to register pg module: %v", err)
	}

	// Test creating a client and using escape functions
	result, err := runtime.RunString(`
		var client = new pg.Client({});
		var escapedId = client.escapeIdentifier('user_name');
		var escapedLiteral = client.escapeLiteral("John's data");
		JSON.stringify({
			identifier: escapedId,
			literal: escapedLiteral
		});
	`)
	if err != nil {
		t.Fatalf("Failed to test escape functions: %v", err)
	}

	resultStr := result.String()
	if resultStr == "" {
		t.Fatal("Escape functions should return non-empty results")
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
