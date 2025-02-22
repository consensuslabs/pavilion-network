# Configuration Management

## Overview

The Pavilion Network backend uses a flexible configuration system that supports multiple environments and configuration sources. The configuration system is built using Viper and supports both YAML files and environment variables.

## Configuration Files

The system uses two main configuration files:

- `config.yaml`: Main configuration file for development/production
- `config_test.yaml`: Configuration file for testing environment

## Environment Variables

Environment variables are loaded from:

- `.env`: For development/production environments
- `.env.test`: For test environment

## Configuration Structure

### Core Configuration Types

```go
type Config struct {
    Environment string             // Current environment (development, test, production)
    Server      ServerConfig       // Server settings
    Database    DatabaseConfig     // Database connection settings
    Redis       RedisConfig        // Redis settings
    Storage     StorageConfig      // File storage settings
    Logging     LoggingConfig      // Logging configuration
    Ffmpeg      FfmpegConfig      // FFmpeg settings
    Video       VideoConfig        // Video handling settings
    Auth        AuthConfig         // Authentication settings
}
```

### Environment-Specific Settings

The configuration system automatically loads the appropriate configuration based on the `ENV` environment variable:

- When `ENV=test`: Loads `config_test.yaml` and `.env.test`
- Otherwise: Loads `config.yaml` and `.env`

### Configuration Sections

1. **Server Configuration**
   - Port settings
   - Server-specific parameters

2. **Database Configuration**
   - Connection parameters
   - Pool settings
   - SSL mode
   - Timezone

3. **Redis Configuration**
   - Connection address
   - Authentication
   - Database selection

4. **Storage Configuration**
   - Upload directory
   - Temporary directory
   - IPFS settings
   - S3 settings
     - Endpoint
     - Bucket configuration
     - Access credentials
     - Directory structure

5. **Logging Configuration**
   - Log level
   - Format (JSON/Console)
   - Output destination
   - File logging options
   - Development mode
   - Sampling settings

6. **Video Configuration**
   - Size limits
   - Title constraints
   - Description limits
   - Allowed formats

7. **Authentication Configuration**
   - JWT settings
   - Token TTL
   - Secret key management

## Environment Variable Overrides

The following environment variables can override configuration values:

```
LOG_LEVEL              -> logging.level
LOG_FORMAT             -> logging.format
LOG_OUTPUT             -> logging.output
LOG_FILE_ENABLED       -> logging.file.enabled
LOG_FILE_PATH          -> logging.file.path
LOG_FILE_ROTATE        -> logging.file.rotate
LOG_FILE_MAX_SIZE      -> logging.file.maxSize
LOG_FILE_MAX_AGE       -> logging.file.maxAge
LOG_ENV_DEVELOPMENT    -> logging.development
LOG_SAMPLING_INITIAL   -> logging.sampling.initial
LOG_SAMPLING_THEREAFTER-> logging.sampling.thereafter
DB_PASSWORD            -> database.password
S3_ACCESS_KEY_ID      -> storage.s3.accessKeyId
S3_SECRET_ACCESS_KEY  -> storage.s3.secretAccessKey
JWT_SECRET            -> auth.jwt.secret
```

## Default Values

The configuration system sets sensible defaults for various settings:

```go
server.port: 8080
database.sslmode: "disable"
database.timezone: "UTC"
database.pool.maxOpen: 100
database.pool.maxIdle: 10
storage.uploadDir: "uploads"
storage.tempDir: "temp"
redis.addr: "localhost:6379"
redis.db: 0
video.maxSize: 1GB
video.minTitleLength: 3
video.maxTitleLength: 100
video.maxDescLength: 5000
logging.level: "info"
logging.format: "json"
logging.output: "stdout"
```

## Configuration Loading Process

1. Environment Detection
   - Check `ENV` environment variable
   - Select appropriate configuration file
   - Load corresponding `.env` file

2. Default Value Setting
   - Set default values for all configuration parameters
   - Configure environment variable mappings

3. Configuration Loading
   - Read YAML configuration file
   - Apply environment variable overrides
   - Validate configuration values

4. Path Resolution
   - Convert relative paths to absolute paths
   - Ensure storage directories exist

## Test Configuration Management

### Overview

The test configuration system ensures proper isolation between test and production environments, particularly for integration testing with CockroachDB. This separation is crucial for maintaining clean, isolated test environments and catching database-specific issues.

### Test Database Setup

1. **Database Creation**
   ```sql
   CREATE DATABASE pavilion_test;
   ```

2. **Access Methods**
   - Inside CockroachDB container:
     ```bash
     cockroach sql --insecure --host=localhost:26257
     ```
   - From host machine:
     ```bash
     docker exec -it <container_id> cockroach sql --insecure
     ```

3. **Verification**
   ```sql
   SHOW DATABASES;
   ```

### Test Environment Configuration

1. **Configuration Files**
   - `config_test.yaml`: Contains test-specific settings
     - Test database name (`pavilion_test`)
     - Test-specific logging levels
     - Test timeouts and parameters

2. **Environment Variables**
   - `.env.test`: Contains test-specific sensitive information
   - Set `ENV=test` when running tests

3. **Database Management**
   - Automated migration execution
   - Database cleanup between test runs
   - Isolation from production data

### Running Integration Tests

```bash
# Run all tests with test configuration
ENV=test go test ./...

# Run specific package tests
ENV=test go test ./internal/config -v
```

## Best Practices

1. **Security**
   - Never commit sensitive values in configuration files
   - Use environment variables for secrets
   - Keep different environments isolated

2. **Environment Separation**
   - Use `config_test.yaml` for testing
   - Maintain separate databases for different environments
   - Use appropriate logging levels per environment

3. **Validation**
   - All configurations are validated during loading
   - Required fields are checked
   - Port numbers and other numeric values are validated

4. **Logging**
   - Configuration loading is logged
   - Errors during loading are properly reported
   - Success/failure status is tracked

## Usage Example

```go
// Create a logger
logger := logger.NewLogger(loggerConfig)

// Create config service
configService := config.NewConfigService(logger)

// Load configuration
cfg, err := configService.Load(".")
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Use configuration
server := NewServer(cfg)
server.Start()
``` 