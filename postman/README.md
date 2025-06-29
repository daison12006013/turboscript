# TurboScript Postman API Contract

This directory contains a comprehensive Postman collection and environment for testing the TurboScript API. The contract includes dynamic data generation, authentication flows, and comprehensive test scripts that validate the entire API workflow.

## 📁 Files

- **`TurboScript-API.postman_collection.json`** - Complete Postman collection with 10 test scenarios
- **`TurboScript.postman_environment.json`** - Environment variables and configuration
- **`../scripts/run-postman-contract.sh`** - Automated runner script for CI/CD integration

## ✨ Features

### 🔄 Dynamic Data Generation

- **Dynamic Email**: Automatically generates unique emails using timestamps
- **Environment Variables**: All test data is managed through environment variables
- **Unique Test Runs**: Each execution creates fresh test data to avoid conflicts

### 🔐 Authentication Flow

- **JWT Token Management**: Automatically captures and stores authentication tokens
- **Bearer Token Auth**: Uses token-based authentication for protected endpoints
- **Token Lifecycle**: Tests login, token usage, and logout flows

### 📋 Test Scenarios

1. **Health Check** - Validates API is running
2. **User Registration** - Creates new user with dynamic email
3. **User Login** - Authenticates and captures JWT token
4. **Authenticated User Retrieval** - Tests protected endpoint access
5. **Paginated Users** - Tests list endpoints with authentication
6. **Password Change** - Tests password update functionality
7. **Re-authentication** - Validates login with new password
8. **Logout** - Tests token invalidation
9. **Unauthorized Access** - Validates security protection
10. **Invalid Endpoints** - Tests error handling

### 🧪 Comprehensive Testing Scripts

Each request includes detailed test scripts that validate:

- Response status codes
- Response structure and content
- Authentication token handling
- Data integrity and consistency
- Error handling and security

## 🚀 Quick Start

### Option 1: Import into Postman App

1. **Open Postman**
2. **Import Collection**:
   - Click "Import" in Postman
   - Select `TurboScript-API.postman_collection.json`
3. **Import Environment**:
   - Click on "Environments" in the sidebar
   - Click "Import" and select `TurboScript.postman_environment.json`
4. **Select Environment**:
   - Choose "TurboScript Environment" from the environment dropdown
5. **Run Collection**:
   - Right-click on the collection
   - Select "Run collection"
   - Click "Run TurboScript API Contract"

### Option 2: Command Line with Newman

#### Install Newman

```bash
# Install Newman globally
npm install -g newman newman-reporter-html

# Or use the provided script
make postman-install
```

#### Run Tests

```bash
# Manual execution (server must be running)
make up  # Start TurboScript server
make postman-test

# Automatic execution (handles server lifecycle)
make postman-test-auto

# Complete test suite including Postman contract
make test-full-contract
```

## 🛠️ Configuration

### Environment Variables

The Postman environment includes these key variables:

| Variable | Description | Default Value |
|----------|-------------|---------------|
| `base_url` | TurboScript API base URL | `http://localhost:7890` |
| `auth_token` | JWT authentication token | (auto-generated) |
| `test_email` | Dynamic test email | (auto-generated) |
| `test_password` | Test user password | `SecureP@ss123!` |
| `user_uid` | User unique identifier | (auto-generated) |
| `timestamp` | Unique timestamp | (auto-generated) |

### Customizing the Base URL

To test against a different server:

**In Postman:**

1. Go to "TurboScript Environment"
2. Update the `base_url` value

**Command Line:**

```bash
BASE_URL=http://staging.example.com make postman-test
```

## 📊 Test Reports

The Newman runner generates detailed reports:

### HTML Report

- **Location**: `postman/reports/report_TIMESTAMP.html`
- **Features**: Interactive test results, request/response details, test summaries
- **Auto-open**: On macOS, reports open automatically in your browser

### JSON Report

- **Location**: `postman/reports/report_TIMESTAMP.json`
- **Features**: Machine-readable test results for CI/CD integration
- **Usage**: Can be parsed by other tools or scripts

## 🔧 Advanced Usage

### Custom Scripts

The collection includes several types of scripts:

#### Pre-request Scripts

- Generate dynamic test data
- Set up authentication tokens
- Prepare request parameters

#### Test Scripts

- Validate response structure and content
- Check status codes and headers
- Store data for subsequent requests
- Log test progress and results

### Example Pre-request Script

```javascript
// Generate unique email for this test run
const timestamp = pm.environment.get('timestamp');
const dynamicEmail = `test-user-${timestamp}@turboscript.dev`;
pm.environment.set('test_email', dynamicEmail);
console.log(`🔄 Generated email: ${dynamicEmail}`);
```

### Example Test Script

```javascript
pm.test('Authentication successful', function () {
    pm.response.to.have.status(200);
    const responseJson = pm.response.json();
    pm.expect(responseJson.response).to.have.property('access_token');
});

// Store token for subsequent requests
if (pm.response.code === 200) {
    const token = pm.response.json().response.access_token;
    pm.environment.set('auth_token', token);
}
```

## 🔄 CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run Postman Contract Tests
  run: |
    make up
    sleep 10
    make postman-test
    make down
```

### Custom Integration

```bash
#!/bin/bash
# Start server
make up

# Wait for server to be ready
sleep 10

# Run contract tests
./scripts/run-postman-contract.sh

# Capture exit code
exit_code=$?

# Cleanup
make down

# Exit with original code
exit $exit_code
```

## 🐛 Troubleshooting

### Common Issues

**Server Not Running**

```
Error: Server is not running at http://localhost:7890
Solution: Start the server with `make up`
```

**Newman Not Found**

```
Error: Newman is not installed
Solution: Run `make postman-install` or `npm install -g newman`
```

**Authentication Failures**

- Check that the server is running the latest code
- Verify that user registration is working
- Check server logs: `make logs`

**Test Failures**

- Review the HTML report for detailed error information
- Check server logs for API errors
- Verify database is properly initialized

### Debug Mode

Enable verbose logging:

```bash
# Set environment variable for detailed Newman output
NEWMAN_DEBUG=true make postman-test
```

## 📈 Test Coverage

The Postman contract covers:

- ✅ **Authentication**: Login, logout, token management
- ✅ **User Management**: Registration, profile retrieval, password changes
- ✅ **Data Access**: Paginated lists, filtered queries
- ✅ **Security**: Authorization checks, error handling
- ✅ **API Structure**: Response formats, status codes
- ✅ **Edge Cases**: Invalid endpoints, unauthorized access

## 🤝 Contributing

To add new tests:

1. **Add Request**: Create new request in Postman or edit the JSON file
2. **Add Tests**: Include comprehensive test scripts
3. **Update Environment**: Add any new variables needed
4. **Test Locally**: Run the collection to verify it works
5. **Update Documentation**: Document any new features or variables

## 📋 Best Practices

- **Unique Data**: Always use dynamic data generation to avoid conflicts
- **Cleanup**: The contract is designed to be stateless and cleanup after itself
- **Error Handling**: Include negative test cases and error scenarios
- **Documentation**: Keep test names and descriptions clear and descriptive
- **Environment**: Use environment variables for all configurable values
