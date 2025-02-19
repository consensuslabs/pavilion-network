# Code Organization Guidelines

## Package Structure
Each package follows a consistent structure with clear separation of concerns:

```
backend/
├── internal/                  # Internal packages
│   ├── auth/                 # Authentication package
│   │   ├── handler.go        # HTTP handlers
│   │   ├── interface.go      # Public interfaces
│   │   ├── model.go          # Data models
│   │   ├── service.go        # Business logic
│   │   ├── types.go          # Type definitions
│   │   └── validation.go     # Validation logic
│   │
│   ├── storage/              # Storage package
│   │   ├── interface.go      # Storage interfaces
│   │   ├── types.go          # Shared types
│   │   ├── adapter.go        # Interface adapters
│   │   ├── ipfs/            # IPFS implementation
│   │   │   └── service.go   # IPFS service
│   │   └── s3/              # S3 implementation
│   │       └── service.go   # S3 service
│   │
│   ├── http/                 # HTTP package
│   │   ├── interface.go      # HTTP interfaces
│   │   ├── middleware.go     # HTTP middleware
│   │   ├── response_handler.go # Response handling
│   │   ├── static.go         # Static file serving
│   │   └── types.go          # HTTP types
│   │
│   ├── video/                # Video package
│   │   ├── handler.go        # Video handlers
│   │   ├── interface.go      # Video interfaces
│   │   ├── model.go          # Video models
│   │   ├── service.go        # Video business logic
│   │   ├── types.go          # Video types
│   │   └── validation.go     # Video validation
│   │
│   ├── config/               # Configuration package
│   │   ├── interface.go      # Config interfaces
│   │   ├── service.go        # Config loading
│   │   └── types.go          # Config types
│   │
│   ├── logger/               # Logging package
│   │   ├── interface.go      # Logger interfaces
│   │   ├── service.go        # Logger implementation
│   │   └── types.go          # Logger types
│   │
│   ├── database/             # Database package
│   │   ├── interface.go      # Database interfaces
│   │   └── service.go        # Database service
│   │
│   ├── cache/                # Cache package
│   │   ├── interface.go      # Cache interfaces
│   │   ├── service.go        # Redis implementation
│   │   └── types.go          # Cache types
│   │
│   ├── errors/               # Error package
│   │   ├── constants.go      # Error constants
│   │   ├── errors.go         # Error types
│   │   └── types.go          # Error definitions
│   │
│   └── health/               # Health check package
│       ├── handler.go        # Health handlers
│       └── interface.go      # Health interfaces
│
├── migrations/               # Database migrations
│   ├── main.go              # Migration runner
│   └── 001_create_*.go      # Migration files
│
├── test/                     # Integration and unit tests
├── app.go                    # Application setup
├── main.go                   # Entry point
└── routes.go                 # Route definitions
```

## Package Responsibilities

### Main Package (`backend/`)
- Application initialization and configuration
- Service orchestration
- Route setup
- Graceful shutdown handling

### Auth Package (`internal/auth/`)
- User authentication and authorization
- JWT token management
- Session handling
- User model and operations

### Storage Package (`internal/storage/`)
- File storage abstraction
- IPFS implementation
- S3 implementation
- Storage type definitions
- Interface adapters for compatibility

### HTTP Package (`internal/http/`)
- HTTP response handling
- Middleware functions
- Static file serving
- Common HTTP types and utilities

### Video Package (`internal/video/`)
- Video upload handling
- Video processing
- Transcoding operations
- Video metadata management

### Config Package (`internal/config/`)
- Configuration loading
- Environment variable handling
- Configuration validation
- Type definitions

### Logger Package (`internal/logger/`)
- Logging interface
- Structured logging
- Log level management
- Context-aware logging

### Database Package (`internal/database/`)
- Database connection management
- Migration support
- Common database operations
- Connection pooling

### Cache Package (`internal/cache/`)
- Redis implementation
- Cache interface
- TTL management
- Cache operations

### Errors Package (`internal/errors/`)
- Error type definitions
- Error constants
- Error handling utilities
- Custom error types

### Health Package (`internal/health/`)
- Health check endpoints
- System status monitoring
- Dependency health checks

## Design Principles

### 1. Interface Segregation
- Each package defines its interfaces in `interface.go`
- Interfaces are kept small and focused
- Dependencies are defined through interfaces
- Implementation details are hidden

### 2. Dependency Injection
- Services receive their dependencies through constructors
- No global state
- Easy to test and mock
- Clear dependency chain

### 3. Error Handling
- Consistent error types
- Proper error wrapping
- Contextual error information
- Clear error messages

### 4. Configuration
- Type-safe configuration
- Environment variable support
- Validation at startup
- Sensible defaults

### 5. Adapters
- Interface adapters for compatibility
- Clean integration between packages
- Minimal dependencies
- Clear separation of concerns

## Best Practices

1. **Package Organization**
   - Keep packages focused and small
   - Clear separation of concerns
   - Consistent file naming
   - Proper interface definitions

2. **Error Handling**
   - Use custom error types
   - Include context in errors
   - Proper logging
   - Clean error messages

3. **Configuration**
   - Type-safe configuration
   - Environment variables
   - Validation
   - Documentation

4. **Testing**
   - Unit tests for each package
   - Integration tests
   - Mock interfaces
   - Test coverage

5. **Documentation**
   - Clear package documentation
   - Interface documentation
   - Example usage
   - Clear comments

## Implementation Guidelines

1. **File Size**
   - Keep files under 500 lines
   - Split large files
   - One concept per file
   - Clear file purpose

2. **Dependencies**
   - Minimize external dependencies
   - Version control
   - Security updates
   - Proper vendoring

3. **Logging**
   - Structured logging
   - Proper log levels
   - Context information
   - Performance consideration

4. **Security**
   - Proper authentication
   - Authorization checks
   - Input validation
   - Secure defaults

## Maintenance

- Regular dependency updates
- Security patches
- Performance monitoring
- Code reviews
- Documentation updates 