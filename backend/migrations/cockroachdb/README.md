# CockroachDB Migrations

This directory contains migrations for CockroachDB. These migrations will be executed when the `FORCE_MIGRATION` environment variable or config option is set to `true`.

## Current Migrations

1. **Test Migration** (`001_test_migration.go`) - A simple test migration that creates a table called `migration_test` and inserts a record to verify the migration system works.

## Running Migrations

The migrations will automatically run during application startup when `FORCE_MIGRATION=true` is set in your environment or config file.

### Running Migrations in Development

1. Ensure CockroachDB is running
2. Set `FORCE_MIGRATION=true` in your environment or config file
3. Start the application

### Running Migrations in Production

1. **IMPORTANT**: Always create a backup of your database before running migrations
2. Set `FORCE_MIGRATION=true` in your environment or config file
3. Start the application
4. After the migration completes successfully, set `FORCE_MIGRATION=false`

## Adding New Migrations

To add a new migration:

1. Create a new file in this directory with a sequential number prefix (e.g., `002_your_migration_name.go`)
2. Implement the migration logic
3. Add the migration to the list of migrations in `migrations.go`

## Migration Best Practices

1. Always create backups before modifying schema or data
2. Make migrations idempotent (safe to run multiple times)
3. Handle errors gracefully and provide helpful log messages
4. Test migrations thoroughly in a development environment before running in production 