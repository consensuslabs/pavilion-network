# Pavilion Network Notification System

## 1. Overview
The Pavilion Network notification system is designed to provide real-time notifications for various events occurring within the platform. It follows a domain-driven design approach with clear separation of concerns and standardized interfaces.

## 2. Architecture

### 2.1 Components
1. **Event Streaming**: Apache Pulsar for reliable, scalable event distribution
2. **Notification Service**: Core service for processing events and managing notifications
3. **Hybrid Storage**: Redis for recent notifications + ScyllaDB for historical data
4. **Delivery Mechanisms**: WebSocket for real-time delivery and REST API for history

### 2.2 Flow Diagram
```
┌────────────┐    ┌─────────┐    ┌──────────────┐    ┌──────────────┐
│ Application │───►│ Producer │───►│ Apache Pulsar │───►│  Consumers   │
└────────────┘    └─────────┘    └──────────────┘    └──────┬───────┘
                                                            │
                                                            ▼
┌────────────┐    ┌─────────────┐    ┌──────────────┐    ┌──────────────┐
│   Client    │◄───│ WebSocket/  │◄───│ Notification  │◄───│  Repository  │
└────────────┘    │  REST API   │    │   Service     │    └──────────────┘
                  └─────────────┘    └──────────────┘
```

## 3. Event Types

### 3.1 Video Events
- **VideoUploaded**: When a new video is uploaded
- **VideoLiked**: When a video receives a like
- **VideoShared**: When a video is shared
- **VideoCommented**: When a video receives a comment

### 3.2 Comment Events
- **CommentCreated**: When a comment is posted
- **CommentReplied**: When a reply is added to a comment
- **CommentLiked**: When a comment receives a like

### 3.3 User Events
- **UserFollowed**: When a user is followed
- **UserMentioned**: When a user is mentioned in a comment or video description
- **UserTagged**: When a user is tagged in a video

## 4. Technical Implementation

### 4.1 Configuration
The notification system uses a structured configuration approach with the `ServiceConfig` type:

```go
type ServiceConfig struct {
    // General settings
    Enabled bool
    
    // Pulsar connection settings
    BrokerURL     string
    OperationTimeout time.Duration
    ConnectionTimeout time.Duration
    
    // Security settings
    TLSEnabled bool
    TLSCertPath string
    TLSKeyPath string
    
    // Topic configuration
    Topics struct {
        VideoEvents   string
        CommentEvents string
        UserEvents    string
        DeadLetter    string
        RetryQueue    string
    }
    
    // Retention settings
    RetentionDays int
    
    // Deduplication settings
    DeduplicationEnabled bool
    DeduplicationWindow time.Duration
    
    // Resilience settings
    MaxRetries     int
    RetryDelay     time.Duration
    BackoffFactor  float64
}
```

### 4.2 Error Handling
The system uses standardized error types for consistent error handling:

```go
var (
    ErrNotificationNotFound = errors.New("notification not found")
    ErrInvalidNotification  = errors.New("invalid notification")
    ErrInvalidEventType     = errors.New("invalid event type")
    ErrServiceDisabled      = errors.New("notification service is disabled")
    ErrConnectionFailed     = errors.New("failed to connect to message broker")
)

type NotificationError struct {
    Op      string
    Err     error
    Context map[string]interface{}
}
```

### 4.3 Utility Functions
The system includes utility functions for common operations:

```go
// TruncateContent truncates a string to the specified length and adds ellipsis if needed
func TruncateContent(content string, maxLength int) string {
    if len(content) <= maxLength {
        return content
    }
    return content[:maxLength] + "..."
}
```

### 4.4 Producers
Producers are responsible for publishing events to Pulsar topics:

```go
type Producer interface {
    PublishEvent(ctx context.Context, event Event) error
    Close() error
}
```

### 4.5 Consumers
Consumers process events from Pulsar topics and create notifications:

```go
type Consumer interface {
    Start(ctx context.Context) error
    Stop() error
}
```

### 4.6 Storage
Hybrid storage approach:
- Redis: Recent notifications (7 days)
- ScyllaDB: Historical notifications

#### 4.6.1 ScyllaDB Configuration
```go
// ScyllaDB settings for the notification repository
type ScyllaDBConfig struct {
    // Connection pool settings
    MaxConnections     int           // Maximum number of connections in the pool
    MaxIdleConnections int           // Maximum number of idle connections in the pool
    ConnectTimeout     time.Duration // Timeout for establishing database connections
    Timeout            time.Duration // Timeout for database operations
    
    // Retry settings
    MaxRetries         int           // Maximum number of retry attempts
    RetryInterval      time.Duration // Base interval between retry attempts
}
```

#### 4.6.2 Connection Pooling
The system maintains a pool of ScyllaDB connections to improve performance:
- Multiple simultaneous connections reduce request latency
- Connection reuse avoids the overhead of establishing new connections
- Pool size is configurable to match system resources

#### 4.6.3 Retry Mechanism
Database operations are automatically retried when transient errors occur:
- Intelligent classification of errors as retryable vs. non-retryable
- Exponential backoff to prevent overwhelming the database during outages
- Configurable retry limits and intervals

### 4.7 Delivery
- WebSocket for real-time notifications
- REST API for notification history and management

## 5. Performance Considerations

### 5.1 Scalability
- Horizontal scaling of consumers
- Partitioned topics for parallel processing
- Redis caching for frequently accessed data

### 5.2 Resilience
- Dead Letter Queue (DLQ) for failed messages
- Retry mechanism with exponential backoff
- Circuit breaker for external dependencies
- Intelligent error classification for database operations
- Connection pooling for improved database resilience

#### 5.2.1 Error Classification
The system classifies database errors to determine appropriate handling:

```go
// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
    // Connection errors, timeouts, and temporary database errors are retryable
    if err == gocql.ErrNoConnections || err == gocql.ErrTimeoutNoResponse || err == gocql.ErrConnectionClosed {
        return true
    }
    
    // Context cancellation or deadline exceeded
    if err == context.DeadlineExceeded || err == context.Canceled {
        return true
    }
    
    // Other errors are considered non-retryable
    return false
}
```

### 5.3 Monitoring
- Prometheus metrics for latency and throughput
- Grafana dashboards for visualization
- Alerting for error rates and consumer lag

## 6. Implementation Phases

### Phase 1: Core Infrastructure (Week 1)
- [x] Set up Pulsar in development environment
- [x] Define domain models and interfaces
- [x] Implement basic producer and consumer

### Phase 2: Storage and Service (Week 2)
- [x] Implement notification repository
- [x] Create notification service
- [x] Set up basic error handling

### Phase 3: Resilience (Week 3)
- [x] Implement retry mechanism
- [x] Add dead letter queue
- [x] Implement deduplication
- [x] Implement consumers with graceful shutdown
- [x] Configure hybrid storage
- [x] Set up Prometheus monitoring and alerting
- [x] Implement ScyllaDB connection pooling
- [x] Add intelligent retry with exponential backoff
- [x] Add error classification (retryable vs. non-retryable)

### Phase 4: Delivery System (Week 4-5)
- [ ] Implement REST API and WebSocket
- [ ] Test client reconnection handling

## 7. Monitoring and Maintenance

### 7.1 Metrics to Track
1. Event processing latency
2. Delivery success rate
3. Consumer lag
4. Retry queue depth
5. DLQ volume

### 7.2 Maintenance Tasks
1. Cleanup old notifications (cron job)
2. Performance tuning
3. Security patch updates
4. Backup Pulsar snapshots weekly

## 8. Security Considerations

### 8.1 Data Protection
1. TLS for Pulsar and WebSocket
2. Encrypted storage for sensitive metadata
3. RBAC for API access

### 8.2 Authentication & Authorization
1. JWT with refresh tokens for WebSocket
2. Rate limiting on API endpoints
3. User consent checks for notifications

### 8.3 Development Environment
```yaml
version: '3'
services:
  pulsar:
    image: apachepulsar/pulsar:4.0.3
    ports:
      - "6650:6650"
      - "8080:8080"
    environment:
      PULSAR_MEM: "-Xms512m -Xmx512m"
    command: "bin/pulsar standalone"
  redis:
    image: redis:6.2
    ports:
      - "6379:6379"
```

## 9. Code Organization
```
backend/
├── internal/
│   ├── notification/
│   │   ├── alias.go         # Type aliases and common imports
│   │   ├── config.go        # Configuration structures
│   │   ├── interfaces.go    # Core interfaces
│   │   ├── service.go       # Service implementation
│   │   ├── util.go          # Utility functions
│   │   ├── types/
│   │   │   ├── error.go     # Error types
│   │   │   └── events.go    # Event definitions
│   │   ├── consumer/
│   │   │   ├── comment_consumer.go
│   │   │   ├── user_consumer.go
│   │   │   └── video_consumer.go
│   │   ├── producer/
│   │   │   ├── comment_producer.go
│   │   │   ├── user_producer.go
│   │   │   └── video_producer.go
│   │   ├── repository/
│   │   ├── delivery/
│   │   │   ├── websocket/
│   │   │   └── rest/
│   │   └── tests/
│   │       ├── unit/
│   │       ├── integration/
│   │       └── e2e/
│   └── common/
```

### 9.1 Key Components

#### Domain Models
```go
type Notification struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Type      EventType
    Content   string
    Metadata  map[string]interface{}
    CreatedAt time.Time
    ReadAt    *time.Time
}
```

#### Producers
```go
type VideoEventProducer struct {
    pulsarClient pulsar.Client
    topic        string
}

func (p *VideoEventProducer) PublishVideoUploaded(ctx context.Context, videoID uuid.UUID) error {
    event := &VideoUploadedEvent{
        ID:             uuid.New(),
        VideoID:        videoID,
        CreatedAt:      time.Now(),
        EventKey:       videoID.String(),
        SequenceNumber: time.Now().UnixNano(),
    }
    return p.publish(ctx, event)
}
```

#### Consumers
```go
func (c *VideoEventConsumer) Start(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        default:
            msg, err := c.consumer.Receive(ctx)
            if err != nil {
                return err
            }
            if c.deduplicator.IsDuplicate(msg) {
                c.consumer.Ack(msg)
                continue
            }
            if err := c.processMessage(ctx, msg); err != nil {
                c.sendToDeadLetterQueue(ctx, msg, err)
                c.consumer.Nack(msg)
                continue
            }
            c.consumer.Ack(msg)
        }
    }
}
```

#### WebSocket Hub
```go
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan Notification
    register   chan *Client
    unregister chan *Client
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            delete(h.clients, client)
        case notif := <-h.broadcast:
            for client := range h.clients {
                if client.userID == notif.UserID {
                    client.send <- serializeNotification(notif)
                }
            }
        }
    }
}
```

## 10. Testing Strategy
1. **Unit Tests**: Core logic (e.g., `HandleNotification`)
2. **Integration Tests**: Pulsar and Redis via Testcontainers
3. **Load Tests**: Simulate high event rates with Locust
4. **Chaos Tests**: Validate retries and DLQ under failure

## 11. Deployment
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: notification-service
        image: pavilion/notification-service:latest
```

## 12. Future Enhancements

### 12.1 Features
1. ML-based relevance scoring
2. Notification grouping and prioritization

### 12.2 Scalability
1. Dynamic consumer scaling
2. Sharded storage with custom shard selector

### 12.3 Resilience
1. Circuit breakers for Pulsar
2. Advanced retry policies with backoff

## 13. Conclusion
The notification system combines domain-driven design principles with robust error handling and resilience mechanisms. It provides a scalable and maintainable solution for real-time notifications with clear interfaces and standardized components. The system is designed for future extensibility while maintaining high performance and reliability.