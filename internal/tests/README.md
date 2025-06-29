# TurboScript Tests

This directory contains integration tests, end-to-end tests, and utility test files for TurboScript.

## Test Files

- **`build_test.go`** - Tests for build and compilation functionality
- **`e2e_test.go`** - End-to-end tests for the complete application workflow
- **`integration_test.go`** - Integration tests for component interactions
- **`plugin_test.go`** - Tests for the plugin system functionality
- **`turboplugin_fix_test.go`** - Specific tests for turboPlugin functionality fixes
- **`test_mime.go`** - Utility for testing MIME type detection

## Running Tests

To run all tests in this directory:

```bash
go test ./internal/tests/...
```

To run a specific test file:

```bash
go test ./internal/tests/e2e_test.go
```

To run tests with verbose output:

```bash
go test -v ./internal/tests/...
```

## Test Organization

Tests are organized by functionality:

- **Unit tests** - Located alongside the code they test (e.g., `internal/server/server_test.go`)
- **Integration tests** - Located in this directory
- **End-to-end tests** - Located in this directory
- **Component tests** - Located alongside their respective components

## Coverage

To generate test coverage reports:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```
