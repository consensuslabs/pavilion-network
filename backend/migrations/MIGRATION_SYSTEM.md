# Database Migration System

This document outlines the database migration system for both CockroachDB and ScyllaDB in the Pavilion Network backend.

## Overview

The migration system is designed to support both standard SQL migrations (CockroachDB) and NoSQL migrations (ScyllaDB). The system ensures that migrations run when the `FORCE_MIGRATION=true` environment variable is set, providing a consistent way to manage schema and data changes across different database technologies.

## Directory Structure

```
backend/
├── migrations/
│   ├── main.go                    # Main migration runner for standard SQL migrations
│   ├── cmd/
│   │   └── migrate/               # CLI tool for running standard SQL migrations
│   ├── cockroachdb/               # CockroachDB specific migrations
│   │   ├── migrations.go          # CockroachDB migration runner
│   │   ├── 001_test_migration.go  # Example CockroachDB migration
│   │   └── README.md              # Documentation for CockroachDB migrations
│   └── scylladb/                  # ScyllaDB specific migrations
│       ├── migrations.go          # ScyllaDB migration runner
│       ├── 001_notification_metadata_migration.go  # Notification metadata migration
│       └── README.md              # Documentation for ScyllaDB migrations
└── cmd/
    ├── migrate_cockroachdb/       # Standalone tool for CockroachDB migrations
    │   └── main.go
    └── migrate_scylladb/          # Standalone tool for ScyllaDB migrations
        └── main.go
```

## How Migrations Work

### Automatic Migrations during Application Startup

When the application starts, it performs these steps:

1. Run standard SQL migrations using the main migration system
2. If `FORCE_MIGRATION=true`, run CockroachDB specific migrations
3. If `FORCE_MIGRATION=true`, run ScyllaDB specific migrations
4. If `AUTO_MIGRATE=true`, initialize schemas for ScyllaDB and CockroachDB

### Standalone Migration Commands

You can also run migrations separately using dedicated command-line tools:

- For CockroachDB:
  ```bash
  cd backend
  go build -o migrate_cockroachdb cmd/migrate_cockroachdb/main.go
  ./migrate_cockroachdb
  ```

- For ScyllaDB:
  ```bash
  cd backend
  go build -o migrate_scylladb cmd/migrate_scylladb/main.go
  ./migrate_scylladb
  ```

## Adding New Migrations

### For CockroachDB

1. Create a new Go file in `migrations/cockroachdb/` with a sequential number prefix (e.g., `002_your_migration.go`)
2. Implement the migration logic following the pattern in existing files
3. Add your migration to the `RunMigrations` method in `migrations/cockroachdb/migrations.go`

### For ScyllaDB

1. Create a new Go file in `migrations/scylladb/` with a sequential number prefix (e.g., `002_your_migration.go`)
2. Implement the migration logic following the pattern in existing files
3. Add your migration to the `RunMigrations` method in `migrations/scylladb/migrations.go`

## Best Practices

1. Always create backups before modifying schema or data
2. Make migrations idempotent (safe to run multiple times)
3. Handle errors gracefully and provide helpful log messages
4. Test migrations thoroughly in a development environment before running in production
5. Set `FORCE_MIGRATION=false` after successful migration in production environments 