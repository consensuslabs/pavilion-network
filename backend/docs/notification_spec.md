# Pavilion Network - Notification System Specification

## 1. Overview
This document outlines the technical specification for the Pavilion Network notification system, designed to provide real-time notifications for video, comment, and user interactions using Apache Pulsar and a hybrid storage approach. The system ensures scalability, resilience, and extensibility, with enhancements for production-grade performance and user experience.

## 2. System Architecture

### 2.1 Components
1. **Apache Pulsar**
   - Primary event streaming platform
   - Short-term notification storage (48-hour retention)
   - Real-time notification delivery
2. **Notification Service**
   - Event processing and routing
   - Notification state management
   - Delivery orchestration (WebSocket, push, API)
3. **Storage Layer**
   - Hybrid approach: Pulsar for real-time, ScyllaDB (optional) for historical data
   - Fallback to Pulsar with extended retention if ScyllaDB is unavailable
4. **Resilience Layer**
   - Asynchronous retry queues for failed operations
   - Dead letter queue (DLQ) for unprocessable messages
   - Circuit breakers for external dependencies

### 2.2 Event Types

#### Video Events
- `VIDEO_UPLOADED`: New video upload completed
- `VIDEO_PROCESSED`: Video processing finished
- `VIDEO_UPDATED`: Video metadata updated
- `VIDEO_DELETED`: Video removed

#### Comment Events
- `COMMENT_CREATED`: New comment on video
- `COMMENT_REPLIED`: Reply to comment
- `COMMENT_REACTION`: Reaction added to comment

#### User Events
- `USER_FOLLOWED`: New follower added
- `USER_MENTIONED`: User mentioned in comment
- `AUTH_EVENT`: Security-related notifications

### 2.3 Event Flow
[Source Services] → [Pulsar Topics] → [Notification Service] → [Delivery]
     ↓                   ↓                    ↓                   ↓
Video Service    video-events-topic    Event Processing    Real-time WebSocket
Comment Service  comment-events-topic  State Management    Push Notifications
User Service     user-events-topic     Storage Decisions   API Endpoints
                                            ↓
                                      [Resilience Layer]
                                            ↓
                                     Retry Queue → DLQ

## 3. Technical Implementation

### 3.1 Apache Pulsar Configuration
pulsar:
  url: "pulsar+ssl://localhost:6651"
  topics:
    video_events: "pavilion/notifications/video-events"
    comment_events: "pavilion/notifications/comment-events"
    user_events: "pavilion/notifications/user-events"
    dead_letter: "pavilion/notifications/dead-letter"
    retry_queue: "pavilion/notifications/retry-queue"
  retention:
    time: 48h  # Extendable to 7d with backup strategy
    size: 1024MB
  subscriptions:
    type: "shared"
    name: "notification-processor"
  deduplication:
    enabled: true
    time_window: 2h
  tls:
    cert_file: "/path/to/cert"

### 3.2 Storage Strategy

#### Hybrid Storage Approach
1. **Real-time Layer (Pulsar)**
   - Recent notifications (48 hours default, configurable)
   - Real-time event streaming
   - Temporary storage with retention
2. **Historical Layer (ScyllaDB, Optional)**
   - Long-term storage for notification history
   - Complex query support (e.g., by user, time range)
   - Fallback: Extended Pulsar retention (7 days) if ScyllaDB unavailable

#### Data Consistency Mechanisms
- Idempotent event processing with distributed deduplication (e.g., Redis)
- Transaction log for cross-storage operations
- Periodic reconciliation cron job

### 3.3 Core Service Structure
type NotificationService struct {
    pulsarClient pulsar.Client
    config       *Config
    retryQueue   pulsar.Producer
    dlqProducer  pulsar.Producer
    deduplicator *Deduplicator // Distributed (e.g., Redis-backed)
    wsHub        *websocket.Hub
}

type Config struct {
    PulsarRetentionHours    int
    ArchiveThresholdHours   int
    MaxNotificationsPerUser int
    RetryQueueEnabled       bool
    Resilience              ResilienceConfig
}

type ResilienceConfig struct {
    MaxRetries       int
    BackoffInitial   time.Duration
    BackoffMax       time.Duration
    BackoffFactor    float64
}

type NotificationEvent struct {
    ID             uuid.UUID
    Type           string
    UserID         uuid.UUID
    SourceID       uuid.UUID
    Content        string
    Metadata       map[string]interface{}
    CreatedAt      time.Time
    EventKey       string
    SequenceNumber int64
}

### 3.4 Event Processing Pipeline
func (s *NotificationService) HandleNotification(ctx context.Context, event NotificationEvent) error {
    if err := s.validateEvent(event); err != nil {
        return fmt.Errorf("invalid event: %w", err)
    }

    if s.deduplicator.IsDuplicate(ctx, event.ID.String(), event.EventKey) {
        log.Printf("Duplicate event: %s", event.ID)
        return nil
    }

    if err := s.publishToPulsar(ctx, event); err != nil {
        return s.scheduleRetry(ctx, event, err)
    }

    return s.processForDelivery(ctx, event)
}

func (s *NotificationService) scheduleRetry(ctx context.Context, event NotificationEvent, originalErr error) error {
    if !s.config.RetryQueueEnabled || !s.isRetryableError(originalErr) {
        return s.sendToDeadLetterQueue(ctx, event, originalErr)
    }

    retryMsg := pulsar.ProducerMessage{
        Payload: serializeEvent(event),
        Properties: map[string]string{
            "attempt": "1",
            "max_attempts": strconv.Itoa(s.config.Resilience.MaxRetries),
            "backoff": s.config.Resilience.BackoffInitial.String(),
        },
    }
    _, err := s.retryQueue.Send(ctx, &retryMsg)
    return err
}

### 3.5 API Endpoints
notifications := r.Group("/api/v1/notifications")
{
    notifications.GET("/", authMiddleware(), h.ListNotifications)
    notifications.GET("/unread-count", authMiddleware(), h.GetUnreadCount)
    notifications.PUT("/:id/read", authMiddleware(), h.MarkAsRead)
    notifications.PUT("/read-all", authMiddleware(), h.MarkAllAsRead)
}

### 3.6 WebSocket Authentication
func (h *WebSocketHandler) ServeWS(c *gin.Context) {
    userID, err := h.authenticateUser(c) // JWT with refresh support
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("Failed to upgrade: %v", err)
        return
    }

    client := &Client{
        hub:    h.hub,
        conn:   conn,
        send:   make(chan []byte, 256),
        userID: userID,
    }

    h.hub.register <- client
    go client.writePump()
    go client.readPump()
}

## 4. Performance Considerations

### 4.1 Scalability
1. Horizontal scaling of consumers with dynamic adjustment
2. Topic partitioning based on load
3. Sharded storage for extreme scale

### 4.2 Optimization Strategies
1. **Batching**: Event and delivery batching
2. **Caching**: Redis for recent notifications and unread counts
3. **Grouping**: Consolidate related events for user delivery

### 4.3 Resource Management
1. **Pulsar**: Configurable TTL, consumer group scaling
2. **Storage**: Compression, cleanup cron jobs

### 4.4 Monitoring
var (
    notificationProcessingTime = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "notification_processing_time_seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"event_type"},
    )
    consumerLag = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "consumer_lag_messages",
            Help: "Messages pending in consumer backlog",
        },
        []string{"topic"},
    )
)

## 5. Implementation Phases

### Phase 1: Core Infrastructure (Week 1-2)
- [ ] Set up Pulsar with TLS
- [ ] Configure topics and retention
- [ ] Implement base service with retry queue
- [ ] Set up Redis-backed deduplication
- [ ] Deploy local dev environment (Docker Compose)

### Phase 2: Event Integration (Week 2-3)
- [ ] Integrate video, comment, user services
- [ ] Implement producers with retry logic
- [ ] Add unit tests for producers

### Phase 3: Processing Pipeline (Week 3-4)
- [ ] Implement consumers with graceful shutdown
- [ ] Configure hybrid storage
- [ ] Set up Prometheus monitoring and alerting

### Phase 4: Delivery System (Week 4-5)
- [ ] Implement REST API and WebSocket
- [ ] Add push notification support (e.g., Firebase)
- [ ] Test client reconnection handling

## 6. Monitoring and Maintenance

### 6.1 Metrics to Track
1. Event processing latency
2. Delivery success rate
3. Consumer lag
4. Retry queue depth
5. DLQ volume

### 6.2 Maintenance Tasks
1. Cleanup old notifications (cron job)
2. Performance tuning
3. Security patch updates
4. Backup Pulsar snapshots weekly

## 7. Security Considerations

### 7.1 Data Protection
1. TLS for Pulsar and WebSocket
2. Encrypted storage for sensitive metadata
3. RBAC for API access

### 7.2 Authentication & Authorization
1. JWT with refresh tokens for WebSocket
2. Rate limiting on API endpoints
3. User consent checks for notifications

### 7.3 Development Environment
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

## 8. Code Organization
backend/
├── internal/
│   ├── notification/
│   │   ├── api/
│   │   ├── config/
│   │   ├── domain/
│   │   ├── producer/
│   │   ├── consumer/
│   │   ├── delivery/
│   │   │   ├── websocket/
│   │   │   └── push/
│   │   ├── repository/
│   │   ├── service/
│   │   ├── resilience/
│   │   └── tests/
│   │       ├── unit/
│   │       ├── integration/
│   │       └── e2e/
│   └── common/

### 8.1 Key Components

#### Domain Models
type Notification struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Type      EventType
    Content   string
    Metadata  map[string]interface{}
    CreatedAt time.Time
    ReadAt    *time.Time
}

#### Producers
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

#### Consumers
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

#### WebSocket Hub
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

## 9. Testing Strategy
1. **Unit Tests**: Core logic (e.g., `HandleNotification`)
2. **Integration Tests**: Pulsar and Redis via Testcontainers
3. **Load Tests**: Simulate high event rates with Locust
4. **Chaos Tests**: Validate retries and DLQ under failure

## 10. Deployment
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

## 11. Future Enhancements

### 11.1 Features
1. ML-based relevance scoring
2. Notification grouping and prioritization
3. Multi-device sync with Firebase

### 11.2 Scalability
1. Dynamic consumer scaling
2. Sharded storage with custom shard selector

### 11.3 Resilience
1. Circuit breakers for Pulsar
2. Advanced retry policies with backoff

## 12. Conclusion
The system combines real-time performance with robust resilience and scalability, enhanced by asynchronous retries, detailed testing, and secure deployment options. It’s poised for high-scale notification delivery with a clear path for future growth.