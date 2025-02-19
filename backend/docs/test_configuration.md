# Test Configuration Management for Integration Testing

## Overview

This document outlines how to manage configuration for integration testing, specifically when running tests against a dedicated CockroachDB test database. Separating test configuration from production and development settings ensures a clean, isolated environment and catches database-specific issues.

## Dedicated Test Configuration Files

1. **Test Configuration YAML:**
   - Create a file named `config.test.yaml` in the backend directory. This file should include test-specific settings, such as:
     - Database name: e.g., `pavilion_test` instead of the production `pavilion_db`.
     - Test-specific logging levels, timeouts, and other parameters.

2. **Test Environment Variables:**
   - Create a `.env.test` file to house sensitive information and any overrides needed for tests (for example, `DATABASE_PASSWORD`, `DB_NAME=pavilion_test`, etc.).

3. **Environment Variable Overrides:**
   - Use an environment variable such as `ENV=test` (or a dedicated flag like `TEST_MODE=true`) when running tests. Modify your configuration loader to check for this variable and load `config.test.yaml` (or override settings from the default config) when in test mode.

## Test Database Setup

- **Database Creation:**
  - Ensure your running CockroachDB container has a dedicated test database (e.g., `pavilion_test`). You can manually create this using a CockroachDB shell command or an automated script before tests run.

- **Migration Execution:**
  - In your test setup, programmatically run all migrations using your existing migration tool (e.g., `migrations.RunMigrations(db, "up")`) to prepare the test database with the correct schema.

- **Database Isolation:**
  - Optionally, implement a teardown or cleanup process in your tests to reset the test database between test runs, ensuring that each test starts with a fresh state.

## Running Tests

- **Test Initialization:**
  - In your test setup code, ensure the following:
    - Environment variables are set for testing (e.g., `ENV=test`).
    - Load configuration via your configuration service, which will then load `config.test.yaml` and `.env.test` as needed.
    - Connect to the CockroachDB test database and run migrations.

- **Executing Tests:**
  - You can run your tests with the following command (ensuring the environment is set to test):
    ```
    ENV=test go test ./...
    ```

## Benefits

- **Isolation:** Keeps test data separate, protecting production and development databases.
- **Realistic Conditions:** Running tests against CockroachDB reveals database-specific behaviors that may be missed when using in-memory SQLite tests.
- **Simplified Maintenance:** A clear separation of configuration for different environments minimizes risk and eases future changes.

## Creating the Test Database Using Cockroach SQL

To create the test database in your CockroachDB instance, follow these steps:

1. **Access the Cockroach SQL Shell:**
   - If you are inside the CockroachDB Docker container, run:

     cockroach sql --insecure --host=localhost:26257

   - Alternatively, from your host machine, execute:

     docker exec -it <container_id> cockroach sql --insecure

   Replace `<container_id>` with your Docker container ID or name.

2. **Create the Database:**

   Once in the Cockroach SQL shell, run the following command:

     CREATE DATABASE pavilion_test;

3. **Verify the Creation:**

   To confirm that the database was created successfully, run:

     SHOW DATABASES;

These steps ensure that the `pavilion_test` database is available for your integration tests.

---

By managing configuration separately for tests, you ensure that integration tests are run in an environment that mirrors production as closely as possible, leading to more reliable and robust application behavior upon deployment. 