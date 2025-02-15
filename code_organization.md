# Code Organization Guidelines

## Package Structure
Each package should follow a consistent structure with clear separation of concerns:

```
package/
├── handler.go    (HTTP/API layer)
├── interface.go  (interfaces)
├── model.go      (data models)
├── service.go    (business logic)
├── repository.go (data access, if needed)
└── validation.go (validation logic)
```

## File Naming Conventions
- Use singular form for file names (e.g., `model.go` not `models.go`)
- Be descriptive but concise
- Use lowercase with underscores for multiple words (though single-word filenames are preferred in Go)
- Be consistent across the project

## File Responsibilities

### `handler.go`
- Contains HTTP handlers/controllers
- Handles request/response logic
- Input validation and sanitization
- Route-specific error handling
- No business logic

### `interface.go`
- Contains all interfaces for the package
- If an interface is used only within a package, define it in that package
- If an interface is used across packages, define it in the package that uses it (not the one that implements it)
- Keep interfaces small and focused

### `model.go`
- Contains data models/entities
- Database model definitions
- JSON/serialization tags
- Basic model methods
- No business logic

### `service.go`
- Contains business logic
- Implements interfaces defined in `interface.go`
- Coordinates between different components
- Handles complex operations
- Business rule validation

### `repository.go`
- Contains data access logic
- Database operations
- Query implementations
- No business logic

### `validation.go`
- Contains validation logic
- Input validation rules
- Custom validators
- Validation helper functions

### `type.go` (if needed)
- Contains custom types
- Type definitions
- Type conversion methods
- Configuration structures

## Best Practices

### Single Responsibility Principle
- Each file should have a single, well-defined purpose
- Files should be focused and relatively small (<200 lines)
- Related functionality should be grouped together
- Split large files into smaller, more focused ones

### Interface Design
- Define interfaces where they are used, not where they are implemented
- Keep interfaces small and focused
- Use composition for complex interfaces
- Document interface methods clearly

### Model Organization
- Keep database models in `model.go`
- Separate DTOs (Data Transfer Objects) if needed
- Group related models together
- Include model validation methods where appropriate

### Code Organization Rules
1. No circular dependencies
2. Clear separation of concerns
3. Consistent error handling
4. Proper logging and instrumentation
5. Clear and consistent naming conventions
6. Documentation for public APIs
7. Tests alongside the code they test

## Example Package Structure

### Video Package
```
video/
├── handler.go     (HTTP handlers for video operations)
├── interface.go   (VideoService, IPFSService, etc.)
├── model.go       (Video struct and related models)
├── service.go     (Video business logic)
├── validation.go  (Video validation rules)
└── type.go        (Config and other types)
```

### Auth Package
```
auth/
├── handler.go     (Authentication handlers)
├── interface.go   (AuthService, TokenService, etc.)
├── model.go       (User, Token models)
├── service.go     (Authentication business logic)
└── validation.go  (Auth validation rules)
```

## Implementation Guidelines

1. **Keep Files Small**
   - Aim for <200 lines per file
   - Split functionality when files grow too large
   - Use meaningful file names that reflect content

2. **Clear Dependencies**
   - Explicitly declare dependencies
   - Use dependency injection
   - Avoid global state

3. **Error Handling**
   - Use consistent error types
   - Proper error wrapping
   - Meaningful error messages

4. **Documentation**
   - Document public APIs
   - Include examples where helpful
   - Keep documentation up to date

5. **Testing**
   - Write tests alongside code
   - Follow table-driven test patterns
   - Mock external dependencies

## Maintenance

- Review and update this document as needed
- Ensure new code follows these guidelines
- Refactor existing code gradually to match these patterns
- Discuss and agree on changes to these guidelines with the team 