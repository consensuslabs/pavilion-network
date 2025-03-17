# Notification System Testing Guide

This document provides a guide to test the notification system in the Pavilion Network platform.

## Prerequisites

1. The backend server is running locally at http://localhost:8080
2. You have access to the Swagger UI at http://localhost:8080/swagger/index.html
3. Apache Pulsar is running locally (via Docker)
4. Redis is running locally (via Docker)
5. ScyllaDB is running locally (via Docker)

## Test Scenario 1: Basic API Testing

### Step 1: Register and Authenticate

1. Create a test user account using the `/auth/register` endpoint:
   ```json
   {
     "username": "testuser",
     "email": "test@example.com",
     "password": "password123"
   }
   ```

2. Login with the created credentials using the `/auth/login` endpoint:
   ```json
   {
     "username": "testuser",
     "password": "password123"
   }
   ```

3. Copy the `access_token` from the response for subsequent requests.

### Step 2: Test Notification APIs (Empty State)

1. Use the `GET /api/v1/notifications/` endpoint with authorization header:
   - Header: `Authorization: Bearer <your_access_token>`
   - Expected response: Empty array since no notifications have been created yet

2. Check the unread notification count:
   - Endpoint: `GET /api/v1/notifications/unread-count`
   - Header: `Authorization: Bearer <your_access_token>`
   - Expected response: Count should be 0

### Step 3: Create a Video to Generate Notifications

1. Upload a test video using the `/video/upload` endpoint:
   - Use form-data with:
     - `title`: "Test Notification Video"
     - `description`: "This is a test video to generate notifications"
     - `video`: Upload a sample video file

2. The video upload should trigger a notification via the event system.

### Step 4: Verify Notification Creation

1. Check for new notifications:
   - Endpoint: `GET /api/v1/notifications/`
   - Header: `Authorization: Bearer <your_access_token>`
   - Expected response: Should include at least one VideoUploaded notification

2. Verify unread count increased:
   - Endpoint: `GET /api/v1/notifications/unread-count`
   - Expected response: Count should now be at least 1

### Step 5: Test Mark as Read Functionality

1. Mark a single notification as read:
   - Get a notification ID from the previous response
   - Endpoint: `PUT /api/v1/notifications/:id/read`
   - Replace `:id` with the actual notification ID
   - Header: `Authorization: Bearer <your_access_token>`
   - Expected response: Success message

2. Verify the notification is marked as read:
   - Check the notifications list again
   - The notification should now have a `readAt` timestamp
   - The unread count should have decreased

3. Mark all notifications as read:
   - Endpoint: `PUT /api/v1/notifications/read-all`
   - Header: `Authorization: Bearer <your_access_token>`
   - Expected response: Success message

4. Verify all notifications are marked as read:
   - Unread count should now be 0
   - All notifications in the list should have a `readAt` timestamp

## Test Scenario 2: Testing Comment Notification Integration

### Step 1: Create a Comment to Generate Notifications

1. Add a comment to your video:
   - Endpoint: `POST /video/:id/comment`
   - Replace `:id` with your video ID
   - Request body:
     ```json
     {
       "content": "This is a test comment for notification testing"
     }
     ```

2. This should trigger a CommentCreated notification for the video owner.

### Step 2: Verify Comment Notification

1. Check for new notifications:
   - Endpoint: `GET /api/v1/notifications/`
   - Should now include a CommentCreated notification with appropriate metadata
   - Verify that the comment content is properly truncated using the `TruncateContent` utility function

## Test Scenario 3: Testing Error Handling

### Step 1: Test Service Disabled Error

1. Temporarily disable the notification service in the configuration:
   - Set `Enabled: false` in the notification service configuration
   - Restart the server
   - Attempt to create a notification-triggering event
   - Expected error: `ErrServiceDisabled` ("notification service is disabled")

### Step 2: Test Connection Error

1. Temporarily stop the Pulsar container:
   ```bash
   docker stop pavilion-pulsar
   ```

2. Attempt to create a notification-triggering event
   - Expected error: `ErrConnectionFailed` ("failed to connect to message broker")

3. Restart the Pulsar container:
   ```bash
   docker start pavilion-pulsar
   ```

## Troubleshooting

If you encounter issues during testing:

1. **No notifications appearing:**
   - Check that the notification service is enabled in config.yaml
   - Verify that Pulsar is running and accessible
   - Check the application logs for any errors in notification creation

2. **API access issues:**
   - Ensure your JWT token hasn't expired
   - Refresh your token if needed

3. **Database connection issues:**
   - Verify Redis and ScyllaDB are running
   - Check for any error messages in the application logs

4. **Error handling:**
   - Check for standardized error types in the logs:
     - `ErrNotificationNotFound`
     - `ErrInvalidNotification`
     - `ErrInvalidEventType`
     - `ErrServiceDisabled`
     - `ErrConnectionFailed`

## Verifying Pulsar Integration

To check if notifications are properly flowing through Pulsar:

1. Use the Pulsar admin CLI to check topic statistics:
   ```bash
   docker exec -it pavilion-pulsar bin/pulsar-admin topics stats persistent://pavilion/notifications/video-events
   ```

2. Look for message count increases after triggering actions that should create notifications

## Advanced Testing

For more advanced testing:

1. Create multiple users and test cross-user notifications (follow, comment, etc.)
2. Test high-volume notification scenarios by creating multiple events in quick succession
3. Test notification persistence by restarting the application and verifying notifications are still available
4. Test the retry mechanism by temporarily introducing failures and verifying recovery
5. Test the dead letter queue by causing permanent failures and checking the DLQ