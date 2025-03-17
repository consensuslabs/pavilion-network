# Notification System Testing Plan

## Current Test Coverage

The notification system currently has the following tests:

1. **Repository Tests** (`repository_test.go`):
   - Interface implementation verification
   - Schema manager tests (skipped, requires ScyllaDB)
   - Notification model tests (JSON marshaling/unmarshaling)

2. **Service Tests** (`service_test.go`):
   - Service disabled mode tests
   - Default configuration validation
   - Basic service functionality with default values (skipped, requires Pulsar)

3. **Integration Tests** (`integration_test.go`, `video_integration_test.go`, `comment_integration_test.go`, `user_integration_test.go`):
   - Producer-consumer cycle tests for all event types
   - Full notification creation and retrieval flow
   - Verification of notification content and metadata
   - All tests now passing with proper configuration

4. **Mock Implementations**:
   - `MockRepository`: In-memory repository for testing
   - `MockConsumer`: Test consumer for verifying messages

## System Architecture

The notification system follows a typical event-driven architecture:

1. **Event Production**: Services publish events to Pulsar topics
2. **Event Consumption**: Consumers process events and create notifications
3. **Notification Storage**: ScyllaDB stores notifications with a schema designed for efficient retrieval:
   - `id`: Unique notification ID (UUID)
   - `user_id`: The user receiving the notification (UUID)
   - `type`: Type of notification (TEXT)
   - `content`: Human-readable notification text (TEXT)
   - `metadata`: Additional structured data associated with the notification (MAP<TEXT, TEXT>)
   - `read_at`: Timestamp when notification was read (TIMEUUID, nullable)
   - `created_at`: Timestamp when notification was created (TIMESTAMP)

4. **Repository Layer**:
   - `SaveNotification`: Converts notification objects to database format, handling metadata conversion
   - `GetNotificationsByUserID`: Retrieves notifications with proper type handling and metadata conversion
   - `MarkAsRead`: Updates read status of notifications
   - `GetUnreadCount`: Counts unread notifications for a user

5. **Consumer Layer**:
   - Type-specific consumers (Video, Comment, User) handle different event types
   - Consumers create properly formatted notification content for better user experience
   - Error handling and retries are implemented to ensure reliability

## Testing Gaps and Opportunities

### 1. Unit Tests

#### Missing Unit Tests:
- **Producer Tests**:
  - Test each producer type (Video, Comment, User)
  - Test message creation and properties
  - Test error handling

- **Consumer Tests**:
  - Test message processing for each consumer type
  - Test error handling and retry logic
  - Test dead letter queue functionality
  - Test proper formatting of notification content for all event types

- **Utility Function Tests**:
  - Test `util.TruncateContent`
  - Test `util.GenerateEventID`, `util.GenerateEventTime`, etc.
  - Test `util.CreateProducerMessage`
  - Test content formatting functions

- **Handler Tests**:
  - Test API endpoints for notifications
  - Test pagination, filtering, and sorting
  - Test error responses

### 2. Integration Tests

#### Expand Integration Tests:
- **Complete Producer-Consumer Flow**: ✅ (Implemented)
  - Test all event types (Video, Comment, User) ✅
  - Test event processing and notification creation ✅
  - Test notification retrieval and marking as read ✅
  - Increased wait times for reliable event processing ✅
  - Improved debugging and verification of notification content ✅

- **Database Integration**:
  - Test with various ScyllaDB configurations
  - Test with high volume of notifications
  - Test compatibility with schema changes
  - Test efficient querying using metadata

- **API Integration**:
  - Test notification endpoints with real service
  - Test authentication and authorization
  - Test rate limiting and pagination

### 3. End-to-End Tests

#### New E2E Tests:
- **Full System Flow**:
  - Test notification generation from user actions
  - Test notification delivery to frontend
  - Test real-time updates

- **Cross-Service Integration**:
  - Test interaction with video service
  - Test interaction with user service
  - Test interaction with comment service

### 4. Performance Tests

#### New Performance Tests:
- **Throughput Testing**:
  - Test message processing rate
  - Test notification creation rate
  - Test query performance under load

- **Latency Testing**:
  - Test end-to-end notification delivery time
  - Test database query latency
  - Test API response time

- **Scalability Testing**:
  - Test with increasing number of users
  - Test with increasing number of notifications
  - Test with increasing message rate

### 5. Resilience Tests

#### New Resilience Tests:
- **Fault Tolerance**:
  - Test behavior when Pulsar is down
  - Test behavior when ScyllaDB is down
  - Test recovery after service restart

- **Error Handling**:
  - Test with malformed messages
  - Test with invalid event types
  - Test with database errors

## Implementation Plan

### Phase 1: Unit Tests (1 week)

1. **Week 1: Producer and Consumer Tests**
   - Create unit tests for all producer types
   - Create unit tests for all consumer types
   - Test error handling and edge cases for metadata handling
   - Test notification content formatting

### Phase 2: Integration Tests (1 week)

1. **Week 2: Enhanced Integration Tests**
   - Create tests with high notification volume
   - Test performance of metadata queries
   - Test lifecycle of notifications (creation to read status)

### Phase 3: Performance and Resilience Tests (1 week)

1. **Week 3: Performance Testing**
   - Create throughput tests with the notification schema
   - Create latency tests with metadata filtering
   - Create scalability tests with the repository implementations

### Phase 4: End-to-End Tests (1 week)

1. **Week 4: Full System Flow**
   - Test complete notification flow with frontend integration
   - Test real-time updates with WebSockets
   - Test notification delivery timing

## Integration Test Examples

The following integration test successfully verifies the full notification flow for video uploads:

```go
func TestVideoEventIntegration(t *testing.T) {
    // Set up integration test environment
    _, pulsarClient, scyllaSession, cleanup := SetupIntegrationTest(t)
    defer cleanup()
    
    // Initialize ScyllaDB schema
    err := helpers.InitializeScyllaDBSchema(scyllaSession)
    require.NoError(t, err, "Failed to initialize ScyllaDB schema")
    
    // Create a test service config with unique topics
    serviceConfig := CreateTestServiceConfig()
    
    // Set up notification repository
    repo, err := helpers.CreateNotificationRepository(scyllaSession)
    require.NoError(t, err, "Failed to create notification repository")
    
    // Create test data
    userID := uuid.New()
    videoID := uuid.New()
    title := "Test Integration Video Upload"
    
    // Clean up any existing test data for this user
    err = helpers.CleanupTestData(scyllaSession, userID)
    require.NoError(t, err, "Failed to clean up test data")
    
    // Set up consumer and producer
    videoConsumer := helpers.SetupVideoConsumer(pulsarClient, repo, serviceConfig)
    videoProducer, err := helpers.SetupVideoProducer(pulsarClient, repo, serviceConfig)
    require.NoError(t, err, "Failed to set up video producer")
    
    // Start consumer in background
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go func() {
        consumerErr := videoConsumer.Start(ctx)
        if consumerErr != nil && consumerErr != context.Canceled {
            t.Logf("Consumer error: %v", consumerErr)
        }
    }()
    
    // Publish a video upload event
    err = videoProducer.PublishVideoUploadEvent(context.Background(), userID, videoID, title)
    require.NoError(t, err, "Failed to publish video upload event")
    
    // Wait for event to be processed
    helpers.WaitForProcessing(20 * time.Second)
    
    // Check if notification was created
    exists, notification, err := helpers.CheckNotificationExists(
        context.Background(),
        repo,
        userID,
        string(types.VideoUploaded),
    )
    require.NoError(t, err, "Failed to check if notification exists")
    
    // Verify notification
    assert.True(t, exists, "Notification was not created")
    if exists && notification != nil {
        assert.Equal(t, userID, notification.UserID, "Wrong user ID in notification")
        assert.Equal(t, string(types.VideoUploaded), notification.Type, "Wrong notification type")
        assert.Contains(t, notification.Content, title, "Notification content doesn't contain video title")
        
        // Check metadata
        videoIDFromMetadata, hasVideoID := notification.Metadata["videoId"]
        assert.True(t, hasVideoID, "Missing videoId in notification metadata")
        assert.Equal(t, videoID.String(), videoIDFromMetadata, "Wrong videoId in notification metadata")
    }
}
```

## Future Enhancements

While the current notification system properly stores and retrieves notifications, several enhancements could improve functionality:

1. **Real-Time Push Notifications**: Implement a WebSocket server to push notifications to online users:
   - Maintain the current flow for persistence (Pulsar → ScyllaDB)
   - Add a parallel path for online users (Pulsar → WebSocket → client)
   - Implement user online status tracking

2. **Advanced Filtering**: Enhance API to support filtering by:
   - Notification type
   - Read/unread status
   - Date ranges
   - Content search

3. **Notification Preferences**: Allow users to control which notifications they receive

## Conclusion

The notification system includes a robust event-driven architecture with Pulsar handling message transport and ScyllaDB providing persistent storage. The integration tests now verify the complete flow from event publishing to notification retrieval, with improved data handling for metadata and timestamps.

Key improvements in the system include:
- Structured metadata storage as `map<text, text>` for better querying and representation
- Improved content formatting for all event types, especially for user events like follows/unfollows
- Proper timestamp handling for read/unread status
- Enhanced testing infrastructure with improved wait times and verification

Future work should focus on performance testing, real-time delivery mechanisms, and more comprehensive unit tests to ensure continued reliability as the system evolves. 