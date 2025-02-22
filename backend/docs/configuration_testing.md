# Configuration Testing Guide

## Overview

This document outlines the testing strategy and practices for the Pavilion Network configuration system. The configuration testing ensures that the system correctly loads and validates configuration settings across different environments.

## Test Environment Setup

### Required Files

1. `config_test.yaml`: Test-specific configuration file
2. `.env.test`: Test environment variables file

### Environment Variables

- Set `ENV=test` to activate test configuration
- Test-specific environment variables should be defined in `.env.test`

## Test Coverage

The configuration tests cover:

1. **Environment Detection**
   - Correct loading of test configuration
   - Proper environment variable handling
   - Environment-specific file selection

2. **Configuration Loading**
   - YAML file parsing
   - Environment variable overrides
   - Default value application

3. **Validation**
   - Required field checks
   - Value type validation
   - Path resolution

## Test Implementation

### Mock Logger

```go
type mockLogger struct {
    infoMessages  []string
    errorMessages []string
}

func newMockLogger() *mockLogger {
    return &mockLogger{}
}

func (m *mockLogger) LogInfo(msg string, fields map[string]interface{}) {
    m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) LogError(err error, msg string) error {
    m.errorMessages = append(m.errorMessages, msg)
    return err
}
```

### Test Cases

```go
func TestLoadConfig(t *testing.T) {
    tests := []struct {
        name       string
        env        string
        wantEnv    string
        wantDBName string
    }{
        {
            name:       "Test Environment",
            env:        "test",
            wantEnv:    "test",
            wantDBName: "pavilion_test",
        },
        {
            name:       "Development Environment",
            env:        "development",
            wantEnv:    "development",
            wantDBName: "pavilion_db",
        },
    }
    // ... test implementation
}
```

## Testing Best Practices

1. **Isolation**
   - Use separate test configuration files
   - Don't modify production configurations
   - Clean up test resources after tests

2. **Environment Control**
   - Set environment variables before tests
   - Clean up environment variables after tests
   - Use test-specific paths and resources

3. **Validation Testing**
   - Test both valid and invalid configurations
   - Verify error messages
   - Check default value application

4. **Logging Verification**
   - Verify log messages
   - Check error reporting
   - Validate logging levels

## Running Tests

```bash
# Run all configuration tests
ENV=test go test ./internal/config -v

# Run specific test
ENV=test go test ./internal/config -v -run TestLoadConfig
```

## Test Maintenance

1. **Keeping Tests Updated**
   - Update tests when adding new configuration options
   - Maintain test configuration files
   - Review test coverage regularly

2. **Common Issues**
   - Environment variable conflicts
   - File path resolution in tests
   - Mock logger implementation

3. **Debugging Tests**
   - Use verbose logging in tests
   - Check environment variable state
   - Verify file paths and permissions

## Example: Adding New Configuration Tests

```go
// Testing new configuration option
func TestNewConfigOption(t *testing.T) {
    logger := newMockLogger()
    configService := NewConfigService(logger)

    // Set test environment
    os.Setenv("ENV", "test")
    defer os.Unsetenv("ENV")

    // Load configuration
    cfg, err := configService.Load("../..")
    if err != nil {
        t.Fatalf("Failed to load config: %v", err)
    }

    // Test new option
    if cfg.NewOption != expectedValue {
        t.Errorf("Expected %v, got %v", expectedValue, cfg.NewOption)
    }
}
```

## Integration Testing

1. **Database Integration**
   - Test database connection settings
   - Verify pool configuration
   - Check SSL mode settings

2. **Storage Integration**
   - Test path resolution
   - Verify directory creation
   - Check permissions

3. **Environment Variable Integration**
   - Test override behavior
   - Verify precedence rules
   - Check default fallbacks

## Continuous Integration

1. **CI Pipeline**
   - Run configuration tests in CI
   - Verify test environment setup
   - Check test coverage

2. **Test Reports**
   - Generate test coverage reports
   - Track test execution time
   - Monitor test failures

3. **Environment Management**
   - Manage test environment variables
   - Control test configuration files
   - Handle secrets in CI 