# Pulsar Setup for Notification System

This document provides instructions on setting up and using Apache Pulsar for the Pavilion Network notification system development.

## Overview

The Docker Compose configuration has been updated to include:

- Apache Pulsar standalone node for development and testing
- Initialization script to set up topics, retention policies, and deduplication

## Configuration Details

The Pulsar setup includes:

- **Version**: Apache Pulsar 4.0.3
- **Ports**:
  - `6650`: Pulsar protocol port
  - `6651`: Pulsar SSL protocol port (for future use)
  - `8080`: Pulsar web UI port
- **Topics**:
  - `pavilion/notifications/video-events`: For video-related events
  - `pavilion/notifications/comment-events`: For comment-related events
  - `pavilion/notifications/user-events`: For user-related events
  - `pavilion/notifications/dead-letter`: For unprocessable messages
  - `pavilion/notifications/retry-queue`: For messages to be retried

## Retention and Deduplication

- Default retention policy: 7 days (configurable via `RetentionDays` in `ServiceConfig`)
- Message size retention: 1024 MB (expandable as needed)
- Deduplication is enabled at the namespace level with a configurable window (via `DeduplicationWindow` in `ServiceConfig`)

## Starting the Environment

To start the environment with Pulsar:

```bash
cd backend/docker
docker-compose -f docker-compose-development.yml up -d
```

## Verifying the Setup

You can verify the Pulsar setup by:

1. Accessing the Pulsar web UI at http://localhost:8080
2. Using the Pulsar admin CLI:

```bash
# List namespaces
docker exec -it pavilion-pulsar bin/pulsar-admin namespaces list public

# List topics in the notification namespace
docker exec -it pavilion-pulsar bin/pulsar-admin topics list pavilion/notifications
```

## Connecting to Pulsar from Go Code

The notification system uses a structured configuration approach with the `ServiceConfig` type:

```go
type ServiceConfig struct {
    // General settings
    Enabled bool
    
    // Pulsar connection settings
    BrokerURL         string
    OperationTimeout  time.Duration
    ConnectionTimeout time.Duration
    
    // Security settings
    TLSEnabled  bool
    TLSCertPath string
    TLSKeyPath  string
    
    // Topic configuration
    Topics struct {
        VideoEvents   string
        CommentEvents string
        UserEvents    string
        DeadLetter    string
        RetryQueue    string
    }
    
    // Other settings...
}
```

For local development, you can use:

```go
pulsarClient, err := pulsar.NewClient(pulsar.ClientOptions{
    URL: config.BrokerURL,
    OperationTimeout: config.OperationTimeout,
    ConnectionTimeout: config.ConnectionTimeout,
})
```

For production with TLS:

```go
pulsarClient, err := pulsar.NewClient(pulsar.ClientOptions{
    URL: config.BrokerURL,
    TLSAllowInsecureConnection: false,
    TLSTrustCertsFilePath: config.TLSCertPath,
    OperationTimeout: config.OperationTimeout,
    ConnectionTimeout: config.ConnectionTimeout,
})
```

## Manual Topic Creation

If you need to manually create topics:

```bash
docker exec -it pavilion-pulsar bin/pulsar-admin topics create persistent://pavilion/notifications/my-custom-topic
```

## Troubleshooting

If you encounter issues with the topic initialization:

1. Check if Pulsar is healthy:
   ```bash
   docker exec -it pavilion-pulsar bin/pulsar-admin brokers healthcheck
   ```

2. Run the initialization script manually:
   ```bash
   docker exec -it pavilion-pulsar bash /pulsar/init-pulsar.sh
   ```

3. Check logs:
   ```bash
   docker logs pavilion-pulsar
   docker logs pavilion-pulsar-init
   ```

4. Verify the notification service is enabled in your configuration:
   ```go
   if !config.Enabled {
       return notification.ErrServiceDisabled
   }
   ``` 