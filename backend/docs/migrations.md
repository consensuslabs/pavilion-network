# Database Migration System

## Overview

The Pavilion Network uses a dual-mode migration system that provides flexibility in development while ensuring safety in production. The system is designed to handle both automatic schema updates during development and versioned migrations for production deployments.

## Migration Types

### 1. File-based Migrations (`FORCE_MIGRATION`)
- Located in `backend/migrations/` directory
- Version-controlled and tracked in `schema_migrations` table
- Safe for production use
- Explicit schema changes with Up/Down methods
- Controlled by `FORCE_MIGRATION` environment variable
- Required for production schema changes
- Only affects migrations in the `migrations/` directory

Example migration file (`001_create_video_uploads.go`):
```go
type Migration struct {
    db *gorm.DB
}

func (m *Migration) Up() error {
    return m.db.AutoMigrate(&VideoUpload{})
}

func (m *Migration) Down() error {
    return m.db.Migrator().DropTable(&VideoUpload{})
}
```

### 2. GORM AutoMigrate (`AUTO_MIGRATE`)
- Automatic schema updates based on struct definitions
- Development environment only
- No version tracking or rollback support
- Controlled by `AUTO_MIGRATE` environment variable
- Not recommended for production use
- Updates schema based on Go struct definitions
- Useful for rapid development and prototyping

## Environment Configuration

### Environment Files
1. `.env`: Contains actual configuration values
2. `.env.example`: Template showing required variables

Example configuration:
```bash
# Environment
ENV=development

# Migration Controls
AUTO_MIGRATE=true     # Enable during development
FORCE_MIGRATION=false # Enable when running migrations
```

### Environment Variables

#### `FORCE_MIGRATION`
- Controls execution of versioned migrations
- Default: `false`
- Only affects migrations in `migrations/` directory
- Safe for production use
- Usage:
  ```bash
  # Enable versioned migrations
  FORCE_MIGRATION=true go run .
  ```

#### `AUTO_MIGRATE`
- Controls GORM's automatic schema updates
- Default: `true` in development, `false` otherwise
- Affects automatic schema updates based on structs
- Development only feature
- Usage:
  ```bash
  # Disable automatic migrations
  AUTO_MIGRATE=false go run .
  ```

## Default Behaviors

### Development Environment
```bash
# Default behavior (migrations enabled)
ENV=development go run .

# Disable all migrations
ENV=development AUTO_MIGRATE=false go run .

# Force run versioned migrations
ENV=development FORCE_MIGRATION=true go run .
```

### Production Environment
```bash
# Default behavior (migrations disabled)
ENV=production go run .

# Run versioned migrations
ENV=production FORCE_MIGRATION=true go run .

# AUTO_MIGRATE has no effect in production
ENV=production AUTO_MIGRATE=true go run .  # Migrations still disabled
```

## Migration Tracking

Migrations are tracked in the `schema_migrations` table:

```sql
CREATE TABLE schema_migrations (
    id SERIAL PRIMARY KEY,
    name STRING NOT NULL,
    hash STRING NOT NULL,
    applied_at TIMESTAMP NOT NULL,
    batch_no INT NOT NULL
);
```

Fields:
- `name`: Migration filename
- `hash`: Content hash for integrity checking
- `applied_at`: Execution timestamp
- `batch_no`: Batch identifier for grouped migrations

## Implementation Details

### MigrationConfig

```go
type MigrationConfig struct {
    Environment string
    AutoMigrate bool
    ForceRun    bool
    db          *gorm.DB
}
```

### Migration Decision Logic

```go
func (c *MigrationConfig) ShouldRunMigration() bool {
    // If FORCE_MIGRATION is true, always run migrations
    if c.ForceRun {
        return true
    }

    // In development, check AUTO_MIGRATE
    if c.Environment == "development" || c.Environment == "test"{
        return c.AutoMigrate
    }

    // In production, don't run migrations unless forced
    return false
}
```

## Creating New Migrations

1. Create a new file in `migrations/` following the naming convention:
   ```
   NNN_description.go
   ```
   where NNN is a sequential number (001, 002, etc.)

2. Implement the Migrator interface:
   ```go
   type Migrator interface {
       Up() error
       Down() error
   }
   ```

3. Add migration to the list in `migrations/main.go`

Example:
```go
migrations := []struct {
    Name     string
    Migrator Migrator
}{
    {"001_create_video_uploads.go", NewMigration(db)},
    {"002_create_videos.go", NewVideoMigration(db)},
}
```

## Best Practices

### 1. Development Workflow
- Use `AUTO_MIGRATE=true` during initial development
- Switch to `AUTO_MIGRATE=false` when schema stabilizes
- Create proper versioned migrations for schema changes
- Test migrations with `FORCE_MIGRATION=true`
- Keep migrations small and focused

### 2. Production Deployment
- Never use `AUTO_MIGRATE`
- Always use versioned migrations
- Test migrations in staging first
- Backup database before migrations
- Run during maintenance windows
- Have rollback plan ready
- Only use `FORCE_MIGRATION` for controlled changes

### 3. Migration Design
- Keep migrations small and focused
- Make migrations reversible when possible
- Include both structural and data migrations when needed
- Document complex migrations
- Consider backward compatibility
- Test both Up() and Down() methods

## Troubleshooting

### Common Issues

1. Migrations not running in development
   - Check if `AUTO_MIGRATE=false`
   - Verify environment is set correctly
   - Try `FORCE_MIGRATION=true`
   - Check migration logs

2. Migrations failing in production
   - Check migration logs
   - Verify database connection
   - Check for schema conflicts
   - Ensure migrations are reversible
   - Verify `FORCE_MIGRATION=true`

### Migration Logs

The system provides detailed logging:
```
Migration Configuration:
- Environment: development
- Auto Migrate: true
- Force Migration: false
```

## Security Considerations

1. Production Safety
   - Migrations disabled by default
   - Explicit `FORCE_MIGRATION` required
   - Version tracking for audit
   - Content hashing for integrity
   - No automatic schema changes

2. Database Safety
   - Backup before migrations
   - Transaction support
   - Rollback capabilities
   - Schema validation
   - Migration history tracking

## Migration CLI Tool

Run migrations using the CLI tool:
```bash
# Run migrations up
go run migrations/cmd/migrate/main.go -direction up

# Run migrations down (rollback)
go run migrations/cmd/migrate/main.go -direction down
```

