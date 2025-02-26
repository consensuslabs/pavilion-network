# Video API Test Organization

This document explains the new organized test structure for the video API and provides guidance on migrating tests from the existing monolithic `handler_test.go` file.

## Why Reorganize the Tests?

The current `handler_test.go` file is over 2100 lines long, making it difficult to:
- Navigate and find specific tests
- Maintain and update tests
- Understand test organization
- Make targeted changes without affecting other tests

## New Directory Structure

```
tests/
├── mocks/              # Mock implementations for testing
│   ├── mock_service.go     # Video service mocks
│   ├── mock_storage.go     # Storage service mocks
│   ├── mock_logger.go      # Logger mocks and adapters
│   └── mock_ffmpeg.go      # FFmpeg service mocks
├── helpers/            # Test helper functions
│   └── test_utils.go       # Common test utilities
├── unit/               # Unit tests for specific functionality
│   ├── upload_test.go      # Tests for upload endpoints
│   ├── get_video_test.go   # Tests for getting videos
│   └── list_videos_test.go # Tests for listing videos
└── integration/        # Integration tests
    └── api_test.go         # API endpoint tests
```

## Mock Implementations

The `mocks/` directory contains mock implementations of all interfaces used by the video package. These mocks use the `github.com/stretchr/testify/mock` package to provide test expectations and return values.

### Available Mocks

- `MockVideoService`: Implements the `VideoService` interface
- `MockIPFSService`: Implements the `IPFSService` interface
- `MockStorageService`: Implements the storage service interface
- `MockFFmpegService`: Implements the `ffmpeg.Service` interface
- `MockTempFileManager`: Implements the `tempfile.TempFileManager` interface
- `MockResponseHandler`: Implements the `ResponseHandler` interface
- `MockLogger`: Implements the `Logger` interface
- `TestLoggerAdapter`: Adapts the `testhelper.TestLogger` to our `Logger` interface

## Test Helpers

The `helpers/` directory contains utility functions to simplify writing tests:

- `CreateTestFile`: Creates a temporary file with specific content
- `CreateMultipartRequest`: Creates a multipart HTTP request for file uploads
- `VideoConfigForTest`: Returns a standard video configuration for tests
- `SetupTestVideos`: Creates test video objects
- `SetupTestUploads`: Creates test upload objects
- `SetupTestGinEngine`: Sets up a Gin engine for testing
- `SetupMockServices`: Creates mock services for testing

## Unit and Integration Tests

- **Unit tests**: Focus on testing individual components in isolation
- **Integration tests**: Test how multiple components work together

## Migration Process

Follow these steps to migrate tests from `handler_test.go` to the new structure:

### 1. Identify Test Categories

Group the tests in `handler_test.go` by their functionality:
- Upload-related tests
- Get video tests
- List videos tests
- Status-related tests
- Delete video tests
- Update video tests
- Integration tests

### 2. Move Mock Implementations

1. Extract all mock implementations in `handler_test.go` and move them to the corresponding files in `tests/mocks/`.
2. Update the mock methods to ensure they match the interfaces they implement.
3. Ensure the package names are consistent.

### 3. Extract Helper Functions

1. Identify helper functions like `createTestFile` and `createMultipartRequest`.
2. Move them to `tests/helpers/test_utils.go`.
3. Make them exportable by capitalizing the first letter (e.g., `CreateTestFile`).

### 4. Move Unit Tests

For each group of tests:
1. Create a new file in the `tests/unit/` directory (e.g., `upload_test.go`).
2. Copy the test functions from `handler_test.go` to the new file.
3. Update imports to use the new mock and helper packages.
4. Ensure the tests use the new helper functions.

### 5. Move Integration Tests

1. Move integration tests like `TestVideoEndpoints_Integration` to `tests/integration/api_test.go`.
2. Update imports and function calls to use the new mock and helper packages.

### 6. Update Package Declarations

1. For unit tests, use package `unit`.
2. For integration tests, use package `integration`.
3. For mocks, use package `mocks`.
4. For helpers, use package `helpers`.

### 7. Verify Tests Run Successfully

1. Run the moved tests to ensure they work correctly:
   ```
   go test ./internal/video/tests/...
   ```

2. Verify that tests pass and coverage is maintained.

### 8. Clean Up handler_test.go

Once all tests have been moved and verified:
1. Gradually remove the migrated tests from `handler_test.go`.
2. Keep a minimal set of tests in `handler_test.go` during the transition period to ensure backward compatibility.
3. Eventually, delete `handler_test.go` when all tests have been migrated successfully.

## Example: Migrating an Upload Test

Original in `handler_test.go`:
```go
func TestHandleUpload_Success(t *testing.T) {
    // Test code...
}
```

Migrated to `tests/unit/upload_test.go`:
```go
package unit

import (
    "testing"
    "github.com/consensuslabs/pavilion-network/backend/internal/video/tests/helpers"
    "github.com/consensuslabs/pavilion-network/backend/internal/video/tests/mocks"
    // Other imports...
)

func TestHandleUpload_Success(t *testing.T) {
    // Update test to use helper functions
    mockServices := helpers.SetupMockServices()
    // Rest of the test...
}
```

## Migration Timeline

This refactoring should be done incrementally to minimize risk:

1. **Week 1**: Set up new structure and move mock implementations and helpers
2. **Week 2**: Move unit tests for upload and get endpoints
3. **Week 3**: Move the rest of unit tests
4. **Week 4**: Move integration tests
5. **Week 5**: Clean up and finalize

## Writing New Tests

When writing new tests, follow these guidelines:

1. **For unit tests**: 
   - Add tests to the appropriate file in the `unit/` directory
   - Use mock services to isolate the component being tested
   - Focus on a single aspect of functionality

2. **For integration tests**:
   - Add test cases to the appropriate file in the `integration/` directory
   - Set up realistic mock interactions
   - Test how multiple components interact

## Running Tests

Run all tests:
```
go test ./...
```

Run tests with coverage:
```
go test ./... -cover
```

Run specific test:
```
go test -run TestName
```

## Benefits After Migration

- Focused test files with clear responsibilities
- Better organization for future tests
- Easier maintenance and updates
- Improved readability
- Faster test runs for specific components 