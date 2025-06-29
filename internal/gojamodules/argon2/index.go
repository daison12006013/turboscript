package argon2

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/dop251/goja"
	"golang.org/x/crypto/argon2"
)

// EventLoopRunner represents the interface for running functions on the event loop
type EventLoopRunner interface {
	RunOnLoop(fn func(*goja.Runtime)) bool
}

// Argon2Module represents the argon2 module for goja
type Argon2Module struct {
	runtime *goja.Runtime
	loop    EventLoopRunner
}

// DefaultOptions provides secure defaults following OWASP recommendations
var DefaultOptions = map[string]interface{}{
	"memoryCost":  uint32(65536), // 64MB
	"timeCost":    uint32(3),     // 3 iterations
	"parallelism": uint8(4),      // 4 threads
	"hashLength":  uint32(32),    // 32 bytes
	"saltLength":  16,            // 16 bytes
	"variant":     "argon2id",    // Argon2id variant
}

// New creates a new Argon2 module instance
func New(runtime *goja.Runtime, loop EventLoopRunner) *Argon2Module {
	return &Argon2Module{
		runtime: runtime,
		loop:    loop,
	}
}

// Register registers the argon2 module with the goja runtime
func (a *Argon2Module) Register() error {
	module := a.runtime.NewObject()

	// Async functions
	if err := module.Set("hash", a.hash); err != nil {
		return fmt.Errorf("failed to set hash function: %w", err)
	}
	if err := module.Set("verify", a.verify); err != nil {
		return fmt.Errorf("failed to set verify function: %w", err)
	}

	// Sync functions
	if err := module.Set("hashSync", a.hashSync); err != nil {
		return fmt.Errorf("failed to set hashSync function: %w", err)
	}
	if err := module.Set("verifySync", a.verifySync); err != nil {
		return fmt.Errorf("failed to set verifySync function: %w", err)
	}

	// Defaults
	if err := module.Set("defaults", DefaultOptions); err != nil {
		return fmt.Errorf("failed to set defaults: %w", err)
	}

	return a.runtime.Set("argon2", module)
}

// hash is the async hash function
func (a *Argon2Module) hash(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := a.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("argon2 hash panic: %v", r))
					}
				})
			}
		}()

		if len(call.Arguments) < 1 {
			a.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("hash requires at least 1 argument (password)"))
			})
			return
		}

		password := call.Arguments[0].String()
		if password == "" {
			a.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("password cannot be empty"))
			})
			return
		}

		options := parseOptions(call.Arguments)
		result, err := hashPassword(password, options)
		if err != nil {
			a.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(err)
			})
			return
		}

		a.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(a.runtime.ToValue(result))
		})
	}()

	return a.runtime.ToValue(promise)
}

// verify is the async verify function
func (a *Argon2Module) verify(call goja.FunctionCall) goja.Value {
	promise, resolve, reject := a.runtime.NewPromise()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.loop.RunOnLoop(func(*goja.Runtime) {
					if err, ok := r.(error); ok {
						_ = reject(err)
					} else {
						_ = reject(fmt.Errorf("argon2 verify panic: %v", r))
					}
				})
			}
		}()

		if len(call.Arguments) < 2 {
			a.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(fmt.Errorf("verify requires 2 arguments (hash, password)"))
			})
			return
		}

		hash := call.Arguments[0].String()
		password := call.Arguments[1].String()

		valid, err := verifyPassword(hash, password)
		if err != nil {
			a.loop.RunOnLoop(func(*goja.Runtime) {
				_ = reject(err)
			})
			return
		}

		a.loop.RunOnLoop(func(*goja.Runtime) {
			_ = resolve(a.runtime.ToValue(valid))
		})
	}()

	return a.runtime.ToValue(promise)
}

// hashSync is the synchronous hash function
func (a *Argon2Module) hashSync(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 {
		panic(a.runtime.NewTypeError("hashSync requires at least 1 argument (password)"))
	}

	password := call.Arguments[0].String()
	if password == "" {
		panic(a.runtime.NewTypeError("password cannot be empty"))
	}

	options := parseOptions(call.Arguments)
	result, err := hashPassword(password, options)
	if err != nil {
		panic(a.runtime.NewGoError(err))
	}

	return a.runtime.ToValue(result)
}

// verifySync is the synchronous verify function
func (a *Argon2Module) verifySync(call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 2 {
		panic(a.runtime.NewTypeError("verifySync requires 2 arguments (hash, password)"))
	}

	hash := call.Arguments[0].String()
	password := call.Arguments[1].String()

	valid, err := verifyPassword(hash, password)
	if err != nil {
		panic(a.runtime.NewGoError(err))
	}

	return a.runtime.ToValue(valid)
}

// parseOptions parses the options from function call arguments
func parseOptions(args []goja.Value) map[string]interface{} {
	options := make(map[string]interface{})

	// Copy defaults
	for k, v := range DefaultOptions {
		options[k] = v
	}

	// Parse user options if provided
	if len(args) > 1 && !goja.IsUndefined(args[1]) {
		if optObj := args[1]; optObj != nil {
			if optMap, ok := optObj.Export().(map[string]interface{}); ok {
				for key, value := range optMap {
					switch key {
					case "memoryCost":
						if val := convertToUint32(value); val != 0 {
							options[key] = val
						}
					case "timeCost":
						if val := convertToUint32(value); val != 0 {
							options[key] = val
						}
					case "parallelism":
						if val := convertToUint8(value); val != 0 {
							options[key] = val
						}
					case "hashLength":
						if val := convertToUint32(value); val != 0 {
							options[key] = val
						}
					case "saltLength":
						if val := convertToInt(value); val != 0 {
							options[key] = val
						}
					case "variant":
						if val, ok := value.(string); ok {
							options[key] = val
						}
					}
				}
			}
		}
	}

	return options
}

// Helper functions to convert values from JavaScript to Go types
func convertToUint32(value interface{}) uint32 {
	switch v := value.(type) {
	case float64:
		return uint32(v)
	case int:
		return uint32(v)
	case int32:
		return uint32(v)
	case int64:
		return uint32(v)
	default:
		return 0
	}
}

func convertToUint8(value interface{}) uint8 {
	switch v := value.(type) {
	case float64:
		return uint8(v)
	case int:
		return uint8(v)
	case int32:
		return uint8(v)
	case int64:
		return uint8(v)
	default:
		return 0
	}
}

func convertToInt(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}

// hashPassword hashes a password using Argon2
func hashPassword(password string, options map[string]interface{}) (string, error) {
	memoryCost := options["memoryCost"].(uint32)
	timeCost := options["timeCost"].(uint32)
	parallelism := options["parallelism"].(uint8)
	hashLength := options["hashLength"].(uint32)
	saltLength := options["saltLength"].(int)
	variant := options["variant"].(string)

	// Generate salt
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate hash based on variant
	var hash []byte
	switch variant {
	case "argon2i":
		hash = argon2.Key([]byte(password), salt, timeCost, memoryCost, parallelism, hashLength)
	case "argon2id":
		hash = argon2.IDKey([]byte(password), salt, timeCost, memoryCost, parallelism, hashLength)
	case "argon2d":
		// Go's argon2 package doesn't have Argon2d, fallback to Argon2id
		hash = argon2.IDKey([]byte(password), salt, timeCost, memoryCost, parallelism, hashLength)
	default:
		return "", fmt.Errorf("unsupported variant: %s", variant)
	}

	// Format as encoded hash string
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	result := fmt.Sprintf("$%s$v=19$m=%d,t=%d,p=%d$%s$%s",
		variant, memoryCost, timeCost, parallelism, encodedSalt, encodedHash)

	return result, nil
}

// verifyPassword verifies a password against an Argon2 hash
func verifyPassword(hash, password string) (bool, error) {
	// Parse the hash string: $argon2id$v=19$m=65536,t=3,p=4$salt$hash
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	variant := parts[1]
	version := parts[2]
	params := parts[3]
	encodedSalt := parts[4]
	encodedHash := parts[5]

	// Validate version
	if version != "v=19" {
		return false, fmt.Errorf("unsupported version: %s", version)
	}

	// Parse parameters
	paramMap := make(map[string]int)
	for _, param := range strings.Split(params, ",") {
		kv := strings.Split(param, "=")
		if len(kv) == 2 {
			if val, err := strconv.Atoi(kv[1]); err == nil {
				paramMap[kv[0]] = val
			}
		}
	}

	memoryCost := paramMap["m"]
	timeCost := paramMap["t"]
	parallelism := paramMap["p"]

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(encodedSalt)
	if err != nil {
		return false, fmt.Errorf("invalid salt encoding: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(encodedHash)
	if err != nil {
		return false, fmt.Errorf("invalid hash encoding: %w", err)
	}

	// Generate hash with the same parameters
	var computedHash []byte
	switch variant {
	case "argon2i":
		computedHash = argon2.Key([]byte(password), salt, uint32(timeCost), uint32(memoryCost), uint8(parallelism), uint32(len(expectedHash)))
	case "argon2id", "argon2d": // fallback argon2d to argon2id
		computedHash = argon2.IDKey([]byte(password), salt, uint32(timeCost), uint32(memoryCost), uint8(parallelism), uint32(len(expectedHash)))
	default:
		return false, fmt.Errorf("unsupported variant: %s", variant)
	}

	// Compare hashes in constant time
	if len(computedHash) != len(expectedHash) {
		return false, nil
	}

	var diff byte
	for i := 0; i < len(computedHash); i++ {
		diff |= computedHash[i] ^ expectedHash[i]
	}

	return diff == 0, nil
}
