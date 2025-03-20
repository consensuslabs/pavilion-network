# ScyllaDB Migrations

This directory contains migrations for ScyllaDB tables and data. These migrations will be executed when the `FORCE_MIGRATION` environment variable or config option is set to `true`.

## How to Run the Migration

To run the notification metadata migration (converting from `blob` to `map<text, text>`), follow these steps:

1. Make sure ScyllaDB is running
2. Set `FORCE_MIGRATION=true` in your environment or config.yml file
3. Start your application
4. The migration will run automatically during startup
5. After the migration completes successfully, set `FORCE_MIGRATION=false` to prevent it from running again

## What This Migration Does

The `001_notification_metadata_migration.go` file performs the following actions:

1. Creates a backup of the notifications table with the original schema in `notifications_backup_blob`
2. Alters the metadata column type from `blob` to `map<text, text>`
3. Reads data from the backup table and converts each notification's metadata from JSON blob to a map
4. Updates each notification in the main table with the converted metadata

This migration is idempotent, meaning it's safe to run multiple times. If the schema is already migrated or if the backup already exists, the migration will skip those steps.

## Troubleshooting

If you encounter any issues during migration:

1. Check the application logs for detailed error messages
2. Verify that ScyllaDB is running and accessible
3. Ensure you have appropriate permissions to alter the table schema
4. If the migration fails, the backup table `notifications_backup_blob` will still contain your original data

## Current Migrations

1. **Notification Metadata Migration** (`001_notification_metadata_migration.go`) - Converts the `metadata` field in the notifications table from `blob` to `map<text, text>`. This resolves issues with serialization when handling notification data.
   - Creates a backup of existing data in `notifications_backup_blob`
   - Alters the schema to use `map<text, text>` type
   - Migrates existing data to the new format

## Running Migrations

The migrations will automatically run during application startup when `FORCE_MIGRATION=true` is set in your environment or config file.

### Running Migrations in Development

1. Ensure ScyllaDB is running
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