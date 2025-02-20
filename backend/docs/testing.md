# Testing Guide

## Overview

This document outlines the testing setup, configuration, and procedures for the Pavilion Network backend. We use Go's built-in testing framework along with a custom test helper package to facilitate integration tests with a dedicated test database.

## Test Environment Setup

### Configuration Files

The test environment uses two main configuration files:

1. **config_test.yaml**
   - Located in: `backend/config_test.yaml`
   - Contains test-specific configuration including:
     - Test database connection details
     - Server configuration
     - Storage settings
     - Logging configuration

2. **.env.test**
   - Located in: `backend/.env.test`
   - Contains environment-specific variables:
     ```env
     ENV=test
     AUTO_MIGRATE=true
     FORCE_MIGRATION=true
     ```

### Test Database

The test environment uses a dedicated CockroachDB database (`pavilion_test`) to ensure isolation from development and production data. This database is automatically configured with:
- Separate schema
- Clean state for each test run
- Automatic migrations

## Test Helper Package

The `testhelper` package (`backend/testhelper/`) provides utilities for setting up test environments:

### Key Components

1. **SetupTestDB**
   ```go
   func SetupTestDB(t *testing.T) *gorm.DB
   ```
   - Initializes test database connection
   - Loads test configuration
   - Runs necessary migrations
   - Verifies database connection
   - Returns configured GORM DB instance

2. **LoadTestConfig**
   ```go
   func LoadTestConfig() (*Config, error)
   ```
   - Loads test configuration from `config_test.yaml`
   - Merges with `.env.test` variables
   - Supports override via `TEST_CONFIG_FILE` environment variable

## Running Tests

### Basic Test Execution

To run all tests:
```bash
ENV=test go test ./... -v
```

To run tests for a specific package:
```bash
ENV=test go test ./internal/auth -v
```

### Environment Variables

Key environment variables for testing:
- `ENV=test`: Ensures test configuration is loaded
- `AUTO_MIGRATE=true`: Enables automatic schema migrations
- `FORCE_MIGRATION=true`: Forces migration execution
- `TEST_DB`: (Optional) Override test database name
- `TEST_CONFIG_FILE`: (Optional) Specify custom config file location

### Test Database Setup

Before running tests, ensure:
1. CockroachDB is running
2. Test database exists:
   ```sql
   CREATE DATABASE pavilion_test;
   ```

## Writing Tests

### Integration Tests

Example of an integration test using the test helper:

```go
func TestUserRegistration(t *testing.T) {
    // Setup test database
    db := testhelper.SetupTestDB(t)
    
    // Create service instance
    service := NewService(db)
    
    // Run test
    user, err := service.Register(RegisterRequest{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "password123",
    })
    
    // Assert results
    if err != nil {
        t.Errorf("failed to register user: %v", err)
    }
    if user.Username != "testuser" {
        t.Errorf("expected username 'testuser', got '%s'", user.Username)
    }
}
```

### Best Practices

1. **Database State**
   - Always use `SetupTestDB` for database tests
   - Clean up test data after each test
   - Don't assume database state

2. **Configuration**
   - Use test configuration files
   - Don't modify production/development configs
   - Use environment variables for overrides

3. **Test Structure**
   - Follow Go's testing conventions
   - Use table-driven tests where appropriate
   - Include both positive and negative test cases

4. **Assertions**
   - Use clear error messages
   - Check all relevant fields
   - Test error conditions

## Continuous Integration

Our CI pipeline:
1. Sets up test database
2. Loads test configuration
3. Runs all tests
4. Reports test coverage

## Troubleshooting

Common issues and solutions:

1. **Database Connection Issues**
   - Verify CockroachDB is running
   - Check database exists
   - Confirm connection settings in `config_test.yaml`

2. **Configuration Loading Failures**
   - Verify `.env.test` exists and is accessible
   - Check `config_test.yaml` syntax
   - Confirm environment variables

3. **Migration Failures**
   - Check `AUTO_MIGRATE` and `FORCE_MIGRATION` settings
   - Verify database permissions
   - Review migration logs

## Future Improvements

Planned enhancements to the testing framework:
1. Add test data factories
2. Implement test database cleanup helpers
3. Add performance testing utilities
4. Enhance test coverage reporting 