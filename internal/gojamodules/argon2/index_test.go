package argon2

import (
	"testing"

	"github.com/dop251/goja"
)

// MockEventLoopRunner for testing
type MockEventLoopRunner struct {
	functions []func(*goja.Runtime)
}

func (m *MockEventLoopRunner) RunOnLoop(fn func(*goja.Runtime)) bool {
	m.functions = append(m.functions, fn)
	return true
}

func (m *MockEventLoopRunner) ExecutePending(rt *goja.Runtime) {
	for _, fn := range m.functions {
		fn(rt)
	}
	m.functions = nil
}

func TestArgon2Module_Register(t *testing.T) {
	rt := goja.New()
	loop := &MockEventLoopRunner{}
	module := New(rt, loop)

	err := module.Register()
	if err != nil {
		t.Fatalf("Failed to register module: %v", err)
	}

	// Test if argon2 object exists
	val := rt.Get("argon2")
	if val == nil || goja.IsUndefined(val) {
		t.Fatal("argon2 module not registered")
	}

	obj := val.(*goja.Object)

	// Test if all functions exist
	functions := []string{"hash", "verify", "hashSync", "verifySync", "defaults"}
	for _, fn := range functions {
		if obj.Get(fn) == nil || goja.IsUndefined(obj.Get(fn)) {
			t.Errorf("Function %s not found", fn)
		}
	}
}

func TestHashPassword(t *testing.T) {
	options := map[string]interface{}{
		"memoryCost":  uint32(64),
		"timeCost":    uint32(1),
		"parallelism": uint8(1),
		"hashLength":  uint32(32),
		"saltLength":  16,
		"variant":     "argon2id",
	}

	hash, err := hashPassword("test-password", options)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Fatal("Hash is empty")
	}

	// Test if hash has correct format
	if len(hash) < 50 {
		t.Fatal("Hash too short")
	}

	// Test verification
	valid, err := verifyPassword(hash, "test-password")
	if err != nil {
		t.Fatalf("Failed to verify password: %v", err)
	}

	if !valid {
		t.Fatal("Password verification failed")
	}

	// Test with wrong password
	valid, err = verifyPassword(hash, "wrong-password")
	if err != nil {
		t.Fatalf("Failed to verify wrong password: %v", err)
	}

	if valid {
		t.Fatal("Wrong password should not verify")
	}
}

func TestParseOptions(t *testing.T) {
	rt := goja.New()

	// Test with default options (no arguments)
	args1 := []goja.Value{rt.ToValue("password")}
	options1 := parseOptions(args1)

	if options1["memoryCost"] != uint32(65536) {
		t.Errorf("Expected memoryCost 65536, got %v", options1["memoryCost"])
	}

	// Test with custom options using JavaScript object creation
	script := `({
		memoryCost: 128,
		timeCost: 2,
		variant: "argon2i"
	})`

	val, err := rt.RunString(script)
	if err != nil {
		t.Fatalf("Failed to create options object: %v", err)
	}

	args2 := []goja.Value{rt.ToValue("password"), val}
	options2 := parseOptions(args2)

	// Since we now support both int and float64, this should work
	if options2["memoryCost"] != uint32(128) {
		t.Errorf("Expected memoryCost 128, got %v", options2["memoryCost"])
	}
	if options2["timeCost"] != uint32(2) {
		t.Errorf("Expected timeCost 2, got %v", options2["timeCost"])
	}
	if options2["variant"] != "argon2i" {
		t.Errorf("Expected variant argon2i, got %v", options2["variant"])
	}
}

func TestSynchronousFunctions(t *testing.T) {
	rt := goja.New()
	loop := &MockEventLoopRunner{}
	module := New(rt, loop)

	err := module.Register()
	if err != nil {
		t.Fatalf("Failed to register module: %v", err)
	}

	// Test hashSync
	script := `
		const hash = argon2.hashSync('test-password', {
			memoryCost: 64,
			timeCost: 1,
			parallelism: 1
		});
		hash;
	`

	val, err := rt.RunString(script)
	if err != nil {
		t.Fatalf("Failed to run hashSync: %v", err)
	}

	hash := val.String()
	if hash == "" {
		t.Fatal("hashSync returned empty string")
	}

	// Test verifySync
	verifyScript := `
		const isValidCorrect = argon2.verifySync('` + hash + `', 'test-password');
		isValidCorrect;
	`

	val2, err := rt.RunString(verifyScript)
	if err != nil {
		t.Fatalf("Failed to run verifySync: %v", err)
	}

	if !val2.ToBoolean() {
		t.Fatal("verifySync should return true for correct password")
	}

	// Test with wrong password
	wrongScript := `
		const isValidWrong = argon2.verifySync('` + hash + `', 'wrong-password');
		isValidWrong;
	`

	val3, err := rt.RunString(wrongScript)
	if err != nil {
		t.Fatalf("Failed to run verifySync with wrong password: %v", err)
	}

	if val3.ToBoolean() {
		t.Fatal("verifySync should return false for wrong password")
	}
}
