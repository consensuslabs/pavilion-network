# Comment System V2 Implementation Plan

## Introduction
This document details the technical implementation plan for Pavilion Network's Comment System V2. It focuses on the technical approach, specific code changes, database schema updates, and timelines required to implement the optimizations outlined in the Comment System V2 Specification.

## Current System Architecture
Our current architecture uses:
- ScyllaDB for comment storage with the following tables:
  ```cql
  /* Main comments table */
  CREATE TABLE comments (
    id uuid PRIMARY KEY,
    video_id uuid,
    user_id uuid,
    content text,
    created_at timestamp,
    updated_at timestamp,
    deleted_at timestamp,
    parent_id uuid,
    likes int,
    dislikes int,
    status text
  );

  /* Index table for video comments */
  CREATE TABLE comments_by_video (
    video_id uuid,
    comment_id uuid,
    created_at timestamp,
    PRIMARY KEY (video_id, created_at, comment_id)
  ) WITH CLUSTERING ORDER BY (created_at DESC, comment_id ASC);

  /* Index table for comment replies */
  CREATE TABLE replies (
    parent_id uuid,
    comment_id uuid,
    created_at timestamp,
    PRIMARY KEY (parent_id, created_at, comment_id)
  ) WITH CLUSTERING ORDER BY (created_at DESC, comment_id ASC);

  /* Reactions table */
  CREATE TABLE reactions (
    comment_id uuid,
    user_id uuid,
    type text,
    created_at timestamp,
    PRIMARY KEY (comment_id, user_id)
  );
  ```

## Detailed Implementation Plan

### Phase 1: Foundation (Weeks 1-2)

#### Week 1: Schema Updates and Redis Integration

##### 1.1 Schema Modifications (Days 1-2)
1. Create a new `comment_metrics` table:
   ```cql
   CREATE TABLE comment_metrics (
     comment_id uuid PRIMARY KEY,
     likes counter,
     dislikes counter,
     replies_count counter,
     last_updated timestamp,
     update_frequency text  /* 'realtime', 'high', 'normal', 'low' */
   );
   ```

2. Create a `video_stats` table for aggregated video metrics:
   ```cql
   CREATE TABLE video_stats (
     video_id uuid PRIMARY KEY,
     comments_count counter,
     reactions_count counter,
     last_updated timestamp,
     is_trending boolean,
     update_frequency text
   );
   ```

3. Update existing `reactions` table to better support event sourcing:
   ```cql
   CREATE TABLE reaction_events (
     reaction_id uuid,
     comment_id uuid,
     user_id uuid,
     type text,
     action text, /* 'add' or 'remove' */
     created_at timestamp,
     PRIMARY KEY ((comment_id), created_at, reaction_id)
   ) WITH CLUSTERING ORDER BY (created_at DESC, reaction_id ASC);
   ```

##### 1.2 Redis Integration (Days 3-5)
1. Set up Redis configuration in app config:
   ```go
   // config/app_config.go
   type RedisConfig struct {
     Host     string `yaml:"host"`
     Port     int    `yaml:"port"`
     Password string `yaml:"password"`
     DB       int    `yaml:"db"`
     TTL      int    `yaml:"ttl"` // Default TTL in seconds
   }
   ```

2. Implement Redis service:
   ```go
   // internal/redis/service.go
   type RedisService struct {
     client *redis.Client
     logger *zap.Logger
     config *config.RedisConfig
   }

   func NewRedisService(cfg *config.Config, logger *zap.Logger) (*RedisService, error) {
     // Initialize Redis client
   }

   // Methods for caching and rate limiting
   func (r *RedisService) CacheComments(videoID string, comments []dto.Comment) error
   func (r *RedisService) GetCachedComments(videoID string) ([]dto.Comment, error)
   func (r *RedisService) IncrementReactionCounter(commentID string, reactionType string) error
   func (r *RedisService) GetReactionCounts(commentID string) (map[string]int, error)
   func (r *RedisService) CheckRateLimit(userID string, action string) (bool, error)
   ```

3. Update app initialization to include Redis:
   ```go
   // backend/app.go
   func (a *App) initRedis() error {
     // Initialize Redis service
   }
   ```

##### 1.3 Background Worker Framework (Day 5)
1. Create a worker manager:
   ```go
   // internal/worker/manager.go
   type WorkerManager struct {
     workers  map[string]Worker
     quitChan chan bool
     wg       sync.WaitGroup
     logger   *zap.Logger
   }

   func NewWorkerManager(logger *zap.Logger) *WorkerManager {
     // Initialize worker manager
   }

   // Methods to register and manage workers
   func (w *WorkerManager) RegisterWorker(name string, worker Worker) error
   func (w *WorkerManager) StartAll() error
   func (w *WorkerManager) StopAll() error
   ```

2. Create interface for workers:
   ```go
   // internal/worker/worker.go
   type Worker interface {
     Start() error
     Stop() error
     Name() string
   }
   ```

#### Week 2: Background Reconciliation and Rate Limiting

##### 2.1 Implement Reconciliation Workers (Days 1-3)
1. Create metrics reconciliation worker:
   ```go
   // internal/comment/workers/metrics_worker.go
   type MetricsWorker struct {
     scyllaService *comment.ScyllaService
     redisService  *redis.RedisService
     quitChan      chan bool
     wg            sync.WaitGroup
     logger        *zap.Logger
     config        *config.WorkerConfig
   }

   func NewMetricsWorker(
     scylla *comment.ScyllaService,
     redis *redis.RedisService,
     logger *zap.Logger,
     config *config.WorkerConfig,
   ) *MetricsWorker {
     // Initialize metrics worker
   }

   // Worker methods
   func (w *MetricsWorker) Start() error
   func (w *MetricsWorker) Stop() error
   func (w *MetricsWorker) Name() string
   func (w *MetricsWorker) reconcileCommentMetrics() error
   func (w *MetricsWorker) reconcileVideoStats() error
   ```

2. Implement frequency-based scheduling:
   ```go
   // internal/worker/scheduler.go
   type Scheduler struct {
     tasks    map[string]*Task
     logger   *zap.Logger
     quitChan chan bool
   }

   type Task struct {
     Name     string
     Interval time.Duration
     Fn       func() error
     LastRun  time.Time
   }

   func NewScheduler(logger *zap.Logger) *Scheduler {
     // Initialize scheduler
   }

   // Methods to manage tasks
   func (s *Scheduler) AddTask(name string, interval time.Duration, fn func() error) error
   func (s *Scheduler) Start() error
   func (s *Scheduler) Stop() error
   ```

##### 2.2 Integrate Rate Limiting (Days 4-5)
1. Implement rate limiting middleware:
   ```go
   // internal/middleware/rate_limit.go
   func RateLimitMiddleware(redisService *redis.RedisService, action string, limit int, window time.Duration) gin.HandlerFunc {
     return func(c *gin.Context) {
       // Rate limiting logic
       userID, exists := c.Get("userID")
       if !exists {
         c.Next()
         return
       }

       allowed, err := redisService.CheckRateLimit(userID.(string), action, limit, window)
       if err != nil {
         c.Next() // Fail open if Redis is unavailable
         return
       }

       if !allowed {
         c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
           "error": "Rate limit exceeded. Please try again later.",
         })
         return
       }

       c.Next()
     }
   }
   ```

2. Apply rate limiting to handlers:
   ```go
   // Application of middleware in routes
   router.POST("/video/:id/comment", 
     middleware.AuthMiddleware(authService, responseHandler),
     middleware.RateLimitMiddleware(redisService, "create_comment", 5, time.Minute),
     commentHandler.CreateComment)

   router.POST("/comment/:id/reaction",
     middleware.AuthMiddleware(authService, responseHandler),
     middleware.RateLimitMiddleware(redisService, "add_reaction", 20, time.Minute),
     commentHandler.AddReaction)
   ```

### Phase 2: Event Processing (Weeks 3-4)

#### Week 3: Apache Pulsar Setup and Event Producers

##### 3.1 Pulsar Infrastructure (Days 1-2)
1. Add Pulsar configuration:
   ```go
   // config/app_config.go
   type PulsarConfig struct {
     URL       string `yaml:"url"`
     AdminURL  string `yaml:"admin_url"`
     BatchSize int    `yaml:"batch_size"`
     Topics    struct {
       Comments string `yaml:"comments"`
       Reactions string `yaml:"reactions"`
       Metrics   string `yaml:"metrics"`
     } `yaml:"topics"`
   }
   ```

2. Create Pulsar Docker setup (docker-compose.yml update):
   ```yaml
   pulsar:
     image: apachepulsar/pulsar:2.10.1
     ports:
       - "6650:6650"
       - "8080:8080"
     volumes:
       - pulsar_data:/pulsar/data
       - pulsar_conf:/pulsar/conf
     environment:
       - PULSAR_MEM="-Xms512m -Xmx512m"
     command: bin/pulsar standalone
   ```

##### 3.2 Pulsar Service Integration (Days 3-5)
1. Implement Pulsar client service:
   ```go
   // internal/pulsar/service.go
   type PulsarService struct {
     client      pulsar.Client
     producers   map[string]pulsar.Producer
     logger      *zap.Logger
     config      *config.PulsarConfig
   }

   func NewPulsarService(cfg *config.PulsarConfig, logger *zap.Logger) (*PulsarService, error) {
     // Initialize Pulsar client and service
   }

   // Methods for producing messages
   func (p *PulsarService) CreateProducer(topic string) (pulsar.Producer, error)
   func (p *PulsarService) SendMessage(topic string, key string, message []byte) error
   func (p *PulsarService) SendMessageAsync(topic string, key string, message []byte, callback func(pulsar.MessageID, *pulsar.ProducerMessage, error))
   func (p *PulsarService) Close() error
   ```

2. Define event schemas:
   ```go
   // internal/events/schemas.go
   type CommentEvent struct {
     EventType  string    `json:"event_type"` // "created", "updated", "deleted"
     CommentID  string    `json:"comment_id"`
     VideoID    string    `json:"video_id"`
     UserID     string    `json:"user_id"`
     Content    string    `json:"content,omitempty"`
     ParentID   string    `json:"parent_id,omitempty"`
     CreatedAt  time.Time `json:"created_at"`
     Properties map[string]interface{} `json:"properties,omitempty"`
   }

   type ReactionEvent struct {
     EventType  string    `json:"event_type"` // "added", "removed"
     ReactionID string    `json:"reaction_id"`
     CommentID  string    `json:"comment_id"`
     UserID     string    `json:"user_id"`
     Type       string    `json:"type"` // "like", "dislike"
     CreatedAt  time.Time `json:"created_at"`
   }
   ```

#### Week 4: Event Consumers and Asynchronous Processing

##### 4.1 Event Consumers Implementation (Days 1-3)
1. Implement consumer framework:
   ```go
   // internal/pulsar/consumer.go
   type MessageHandler func(pulsar.Message) error

   type Consumer struct {
     consumer pulsar.Consumer
     handler  MessageHandler
     logger   *zap.Logger
     quitChan chan bool
     wg       sync.WaitGroup
   }

   func NewConsumer(
     client pulsar.Client,
     topic string,
     subscription string,
     handler MessageHandler,
     logger *zap.Logger,
   ) (*Consumer, error) {
     // Initialize consumer
   }

   // Methods for consuming messages
   func (c *Consumer) Start() error
   func (c *Consumer) Stop() error
   ```

2. Implement specific event consumers:
   ```go
   // internal/comment/consumers/comment_consumer.go
   func NewCommentConsumer(
     pulsarClient pulsar.Client,
     scyllaService *comment.ScyllaService,
     redisService *redis.RedisService,
     logger *zap.Logger,
     config *config.PulsarConfig,
   ) (*pulsar.Consumer, error) {
     // Initialize comment consumer
   }

   func handleCommentEvent(msg pulsar.Message, services *ConsumerServices) error {
     // Process comment events
   }

   // Similar structure for reaction_consumer.go
   ```

##### 4.2 Update Handlers for Event-Driven Processing (Days 4-5)
1. Refactor comment handler to use event-driven approach:
   ```go
   // internal/comment/handler.go
   func (h *Handler) CreateComment(c *gin.Context) {
     // Parse request and validate
     // ...

     // Create comment in database
     comment, err := h.commentService.CreateComment(...)
     if err != nil {
       // Handle error
     }

     // Publish event asynchronously
     go func() {
       event := &events.CommentEvent{
         EventType: "created",
         CommentID: comment.ID.String(),
         // ... other fields
       }
       
       if err := h.pulsarService.SendMessage(
         h.config.Pulsar.Topics.Comments,
         comment.ID.String(),
         event,
       ); err != nil {
         h.logger.Error("Failed to publish comment event", zap.Error(err))
       }
     }()

     // Return success response
     c.JSON(http.StatusCreated, comment)
   }

   // Similar updates for other handlers
   ```

### Phase 3: Advanced Optimizations (Weeks 5-6)

#### Week 5: ScyllaDB Optimizations and Read/Write Split

##### 5.1 ScyllaDB Schema Optimizations (Days 1-3)
1. Implement time-bucketing for high-volume videos:
   ```cql
   CREATE TABLE comments_by_video_bucketed (
     video_id uuid,
     time_bucket text, /* Format: YYYY-MM-DD-HH */
     comment_id uuid,
     created_at timestamp,
     PRIMARY KEY ((video_id, time_bucket), created_at, comment_id)
   ) WITH CLUSTERING ORDER BY (created_at DESC, comment_id ASC);
   ```

2. Update repository to use bucketed tables:
   ```go
   // internal/comment/repository.go
   func (r *Repository) GetCommentsByVideoIDBucketed(videoID uuid.UUID, page pagination.Page) ([]model.Comment, error) {
     // Calculate time buckets based on page
     // Query comments from appropriate buckets
   }

   func (r *Repository) getTimeBucket(t time.Time) string {
     return t.Format("2006-01-02-15") // YYYY-MM-DD-HH
   }
   ```

##### 5.2 Read/Write Split Implementation (Days 4-5)
1. Create read models and DTOs:
   ```go
   // internal/comment/dto/read_models.go
   type CommentReadModel struct {
     ID          uuid.UUID `json:"id"`
     VideoID     uuid.UUID `json:"video_id"`
     UserID      uuid.UUID `json:"user_id"`
     Content     string    `json:"content"`
     CreatedAt   time.Time `json:"created_at"`
     UpdatedAt   time.Time `json:"updated_at"`
     ParentID    uuid.UUID `json:"parent_id,omitempty"`
     Likes       int       `json:"likes"`
     Dislikes    int       `json:"dislikes"`
     Status      string    `json:"status"`
     UserReacted bool      `json:"user_reacted"`
     ReactionType string   `json:"reaction_type,omitempty"`
   }
   ```

2. Implement service layer with read/write split:
   ```go
   // internal/comment/service.go
   type CommentService struct {
     repository *Repository
     redis      *redis.RedisService
     // Other dependencies
   }

   // Read operations
   func (s *CommentService) GetCommentsByVideoID(ctx context.Context, videoID uuid.UUID, userID uuid.UUID, page pagination.Page) ([]dto.CommentReadModel, error) {
     // Try to get from cache first
     if cachedComments, err := s.redis.GetCachedComments(videoID.String()); err == nil {
       return cachedComments, nil
     }

     // Fall back to database if not in cache
     comments, err := s.repository.GetCommentsByVideoID(videoID, page)
     if err != nil {
       return nil, err
     }

     // Enhance with reaction data and convert to read models
     readModels := s.enhanceCommentsWithMetrics(comments, userID)

     // Cache for future reads
     go s.redis.CacheComments(videoID.String(), readModels)

     return readModels, nil
   }

   // Write operations
   func (s *CommentService) CreateComment(ctx context.Context, input dto.CreateCommentInput) (uuid.UUID, error) {
     // Direct write to database
     commentID, err := s.repository.CreateComment(input)
     
     // Invalidate cache
     go s.redis.InvalidateVideoCache(input.VideoID.String())
     
     return commentID, err
   }
   ```

#### Week 6: Batching Operations

##### 6.1 In-Memory Accumulators (Days 1-3)
1. Implement reaction accumulator:
   ```go
   // internal/comment/accumulator/reaction_accumulator.go
   type ReactionBatch struct {
     Reactions map[string]map[string]string // commentID -> userID -> reactionType
     mu        sync.Mutex
     batchSize int
     onFlush   func(map[string]map[string]string) error
     ticker    *time.Ticker
     quitChan  chan bool
   }

   func NewReactionBatch(batchSize int, flushInterval time.Duration, onFlush func(map[string]map[string]string) error) *ReactionBatch {
     // Initialize batch
   }

   // Methods for batching
   func (b *ReactionBatch) Add(commentID, userID, reactionType string) error
   func (b *ReactionBatch) Start() error
   func (b *ReactionBatch) Stop() error
   func (b *ReactionBatch) flush() error
   ```

2. Integrate with reaction handler:
   ```go
   // internal/comment/handler.go
   func (h *Handler) AddReaction(c *gin.Context) {
     // Parse and validate request
     
     // Add to batch accumulator
     if err := h.reactionBatch.Add(commentID, userID, reactionType); err != nil {
       h.logger.Error("Failed to add reaction to batch", zap.Error(err))
       // Fall back to direct write if batching fails
       if err := h.commentService.AddReaction(commentID, userID, reactionType); err != nil {
         // Handle error
       }
     }
     
     // Return immediate success
     c.Status(http.StatusAccepted)
   }
   ```

##### 6.2 Batch Processing Integration (Days 4-5)
1. Implement batch consumer in Pulsar:
   ```go
   // internal/pulsar/batch_consumer.go
   type BatchConsumer struct {
     consumer   pulsar.Consumer
     batchSize  int
     timeout    time.Duration
     batchFunc  func([]pulsar.Message) error
     logger     *zap.Logger
     quitChan   chan bool
     wg         sync.WaitGroup
   }

   func NewBatchConsumer(
     client pulsar.Client,
     topic string,
     subscription string,
     batchSize int,
     timeout time.Duration,
     batchFunc func([]pulsar.Message) error,
     logger *zap.Logger,
   ) (*BatchConsumer, error) {
     // Initialize batch consumer
   }

   func (c *BatchConsumer) Start() error
   func (c *BatchConsumer) Stop() error
   func (c *BatchConsumer) processBatch() error
   ```

2. Implement batch processing of metrics updates:
   ```go
   // internal/comment/consumers/metrics_consumer.go
   func processBatchedMetrics(messages []pulsar.Message, services *ConsumerServices) error {
     updates := make(map[string]int) // commentID -> delta
     
     // Aggregate metrics from batch
     for _, msg := range messages {
       // Process each message and update aggregated counts
     }
     
     // Update metrics in a single operation
     for commentID, delta := range updates {
       if err := services.ScyllaService.UpdateCommentMetrics(commentID, delta); err != nil {
         // Handle error
       }
     }
     
     return nil
   }
   ```

### Phase 4: Scaling and Finalization (Weeks 7-8)

#### Week 7: Reaction CRDTs and Monitoring

##### 7.1 CRDT Implementation for Reactions (Days 1-3)
1. Implement a CRDT type for reactions:
   ```go
   // internal/comment/crdt/reaction_set.go
   type ReactionOp struct {
     UserID      string
     CommentID   string
     ReactionType string // "like", "dislike", "none"
     Timestamp   int64
     Add         bool // true = add, false = remove
   }

   type ReactionSet struct {
     ops    map[string]ReactionOp // userID+commentID -> operation
     mu     sync.RWMutex
   }

   func NewReactionSet() *ReactionSet {
     return &ReactionSet{
       ops: make(map[string]ReactionOp),
     }
   }

   // CRDT operations
   func (s *ReactionSet) Add(op ReactionOp) bool
   func (s *ReactionSet) Merge(other *ReactionSet) *ReactionSet
   func (s *ReactionSet) GetReactions(commentID string) map[string]string
   func (s *ReactionSet) CountByType(commentID string) map[string]int
   ```

2. Integrate with repository:
   ```go
   // internal/comment/repository.go
   func (r *Repository) ApplyReactionBatch(ops []crdt.ReactionOp) error {
     // Apply batch of reaction operations to database
   }
   ```

##### 7.2 Monitoring and Alerting (Days 4-5)
1. Implement metric collectors:
   ```go
   // internal/metrics/collectors.go
   func RegisterMetrics(registry prometheus.Registerer) {
     // Register various metrics
     cacheHits := prometheus.NewCounter(prometheus.CounterOpts{
       Name: "comment_cache_hits_total",
       Help: "Total number of comment cache hits",
     })
     
     cacheMisses := prometheus.NewCounter(prometheus.CounterOpts{
       Name: "comment_cache_misses_total",
       Help: "Total number of comment cache misses",
     })
     
     // Register more metrics for different aspects
     // ...
     
     registry.MustRegister(cacheHits, cacheMisses, /* ... */)
   }
   ```

2. Add instrumentation:
   ```go
   // internal/redis/service.go
   func (r *RedisService) GetCachedComments(videoID string) ([]dto.Comment, error) {
     comments, err := r.client.Get(context.Background(), r.commentKey(videoID)).Result()
     if err != nil {
       if err == redis.Nil {
         metrics.CacheMisses.Inc()
         return nil, ErrNotFound
       }
       return nil, err
     }
     
     metrics.CacheHits.Inc()
     // Process result
   }
   ```

#### Week 8: Performance Testing and Documentation

##### 8.1 Performance Testing (Days 1-3)
1. Create load testing scripts:
   ```go
   // tools/loadtest/comment_load.go
   func RunCommentLoadTest(config LoadTestConfig) (*LoadTestResult, error) {
     // Set up test environment
     // Generate synthetic load
     // Measure performance metrics
   }
   ```

2. Create benchmark suite:
   ```go
   // tools/benchmark/comment_bench_test.go
   func BenchmarkCommentCreation(b *testing.B) {
     // Benchmark setup
     b.ResetTimer()
     for i := 0; i < b.N; i++ {
       // Create comment
     }
   }

   func BenchmarkReactionProcessing(b *testing.B) {
     // Benchmark setup
     b.ResetTimer()
     for i := 0; i < b.N; i++ {
       // Process reaction
     }
   }
   ```

##### 8.2 Documentation and Finalization (Days 4-5)
1. API documentation updates
2. Developer guides
3. Final testing and verification
4. Deployment plans and rollout strategy

## Database Schema Changes Summary

### New Tables
1. `comment_metrics`: Stores pre-computed metrics about comments
2. `video_stats`: Stores aggregated statistics about videos
3. `reaction_events`: Event log for all reactions
4. `comments_by_video_bucketed`: Time-bucketed version of comments_by_video

## Configuration Requirements

### Redis Configuration
```yaml
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  ttl: 3600  # 1 hour default TTL
```

### Pulsar Configuration
```yaml
pulsar:
  url: "pulsar://localhost:6650"
  admin_url: "http://localhost:8080"
  batch_size: 100
  topics:
    comments: "persistent://pavilion/comments/events"
    reactions: "persistent://pavilion/reactions/events"
    metrics: "persistent://pavilion/metrics/updates"
```

### Worker Configuration
```yaml
workers:
  metrics_reconciliation:
    enabled: true
    high_frequency_interval: "30s"
    normal_frequency_interval: "5m"
    low_frequency_interval: "30m"
```

## Performance Targets and SLOs
- API response times:
  - p95 < 100ms for read operations
  - p99 < 200ms for write operations
- Redis cache hit rate > 80% for active videos
- Background processing latency < 5 minutes
- Support 1000 concurrent users per video
- Handle 100 comments/second per viral video
- Handle 1000 reactions/second per viral video 