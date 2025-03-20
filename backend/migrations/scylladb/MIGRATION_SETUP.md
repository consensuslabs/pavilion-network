# ScyllaDB Migration Setup Summary

## What We've Done

1. Created a dedicated ScyllaDB migrations framework in the `migrations/scylladb` directory.
2. Implemented a migration for notification metadata (`001_notification_metadata_migration.go`) that:
   - Creates a backup of the notifications table
   - Alters the metadata column from `blob` to `map<text, text>`
   - Migrates existing data to the new format
3. Added a migration runner (`migrations.go`) that can be integrated with the application
4. Integrated with the app's startup process to run when `FORCE_MIGRATION=true`
5. Added a standalone migration tool (optional) in `cmd/migrate_scylladb`

## How to Run Migrations

There are two ways to run the migration:

### Method 1: Using Application with FORCE_MIGRATION (Recommended)

1. Set `FORCE_MIGRATION=true` in your environment or config.yml
2. Start your application normally
3. The migration will run during application startup
4. After migration completes, set `FORCE_MIGRATION=false`

### Method 2: Using Standalone Migration Tool (Optional)

1. Build the standalone tool:
   ```bash
   cd /Users/umitdogan/Dev/pavilion-network/backend
   go build -o migrate_scylladb cmd/migrate_scylladb/main.go
   ```

2. Run the tool:
   ```bash
   ./migrate_scylladb
   ```

3. The tool will load configuration from config.yml or environment variables

## Migration File Structure

The migration framework is designed to be extensible. In the future, if you need to add more migrations:

1. Create a new file with a sequential number prefix (e.g., `002_next_migration.go`)
2. Implement the migration logic following the pattern in `001_notification_metadata_migration.go`
3. Add the migration to the `RunMigrations` method in `migrations.go`

## Verification

You can verify the migration was successful by:

1. Checking the application logs for migration success messages
2. Querying the notifications table to confirm metadata is stored as a map
3. Verifying your application can correctly read and write notifications

For additional troubleshooting, refer to the README.md file in the migrations/scylladb directory. 