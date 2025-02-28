---
# Comments Feature Specification

## Overview

The Comments feature allows users to add, view, edit, and delete comments on videos in the Pavilion Network platform. Unlike other data in the system which is stored in CockroachDB, comments will be stored in ScyllaDB, a high-performance NoSQL database, to better handle the sequential and potentially high-volume nature of comment data.

## Architecture

### Data Flow

1. When a client requests comments for a video:
   - The API first verifies the video exists in CockroachDB
   - If the video exists, it retrieves the comments for that video from ScyllaDB
   - The comments are returned to the client

2. When a client adds a comment:
   - The API first verifies the video exists in CockroachDB
   - If the video exists, it stores the comment in ScyllaDB
   - The new comment is returned to the client

3. When a client deletes a comment:
   - The API first verifies the comment exists in ScyllaDB
   - If the comment exists, it soft deletes the comment by setting the `deleted_at` timestamp
   - The comment is not physically removed from the database

4. When a client reacts to a comment (like/dislike):
   - Update Redis for immediate user feedback
   - Send event to Pulsar for async processing
   - Batch-write to ScyllaDB for persistence

### Multi-Layered Reaction Handling

To efficiently handle high-volume reactions (likes/dislikes) while maintaining data about who reacted:

1. **Redis Layer (Real-time):**
   - Store hash mappings of `comment_id:user_id → reaction_type` for quick lookups
   - Maintain sorted sets of recent reactors per comment
   - Maintain counters for total likes/dislikes per comment
   - Set appropriate TTL for automatic expiration of less active data

2. **Pulsar Layer (Async Processing):**
   - Buffer high-volume reaction events
   - Batch-process reactions for efficient database writes
   - Ensure eventual consistency between Redis and ScyllaDB

3. **ScyllaDB Layer (Persistence):**
   - Store individual reaction records with user identity
   - Store pre-aggregated counts for efficient retrieval
   - Optimize table design for common query patterns

### Database Schema

#### ScyllaDB Tables

1. **Comments Table**

```cql
CREATE TABLE comments (
  id uuid,
  video_id uuid,
  user_id uuid,
  content text,
  created_at timestamp,
  updated_at timestamp,
  deleted_at timestamp,
  parent_id uuid,
  likes counter,
  dislikes counter,
  status text, -- ENUM: 'ACTIVE', 'FLAGGED', 'HIDDEN'
  PRIMARY KEY (video_id, id)
) WITH CLUSTERING ORDER BY (id DESC);
```

2. **Comment Reactions Table**

```cql
CREATE TABLE comment_reactions (
  comment_id uuid,
  user_id uuid,
  reaction_type text, -- ENUM: 'LIKE', 'DISLIKE'
  created_at timestamp,
  updated_at timestamp,
  PRIMARY KEY ((comment_id), user_id)
);
```

3. **User Reactions View**

```cql
CREATE MATERIALIZED VIEW user_reactions AS
  SELECT comment_id, user_id, reaction_type, created_at
  FROM comment_reactions 
  WHERE user_id IS NOT NULL AND comment_id IS NOT NULL
  PRIMARY KEY ((user_id), comment_id);
```

### Enum Types

To ensure type safety and consistency, the following enum types will be used:

1. **CommentStatusType**
   - `ACTIVE`: Normal visible comment
   - `FLAGGED`: Comment flagged for review
   - `HIDDEN`: Comment hidden from general view

2. **ReactionType**
   - `LIKE`: Positive reaction
   - `DISLIKE`: Negative reaction

### Integration Points

1. **Video Service**: The Comments service will interact with the Video service to verify video existence.
2. **Auth Service**: The Comments service will use the Auth service to verify user authentication and authorization.
3. **ScyllaDB**: A new database service will be created to interact with ScyllaDB.
4. **Redis**: Used for caching frequent queries and handling real-time reaction counts.
5. **Pulsar**: Used for asynchronous processing of reaction events.

## API Endpoints

### 1. Get Comments for a Video

```
GET /video/:id/comments?page=1&limit=20
```

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Number of comments per page (default: 20)
- `sort`: Sort order (default: "newest", options: "newest", "oldest", "most_liked")

**Response:**
```json
{
  "status": "success",
  "data": {
    "comments": [
      {
        "id": "comment-id",
        "video_id": "video-uuid",
        "user_id": "user-uuid",
        "user_name": "User Display Name",
        "content": "Comment content",
        "created_at": "timestamp",
        "updated_at": "timestamp",
        "likes": 10,
        "dislikes": 2,
        "status": "ACTIVE",
        "replies_count": 5
      }
    ],
    "total": 100,
    "page": 1,
    "limit": 20
  }
}
```

### 2. Get Replies to a Comment

```
GET /comment/:id/replies?page=1&limit=20
```

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Number of replies per page (default: 20)

**Response:**
```json
{
  "status": "success",
  "data": {
    "replies": [
      {
        "id": "reply-id",
        "video_id": "video-uuid",
        "user_id": "user-uuid",
        "user_name": "User Display Name",
        "content": "Reply content",
        "created_at": "timestamp",
        "updated_at": "timestamp",
        "likes": 5,
        "dislikes": 1,
        "status": "ACTIVE",
        "parent_id": "parent-comment-id"
      }
    ],
    "total": 25,
    "page": 1,
    "limit": 20
  }
}
```

### 3. Add a Comment

```
POST /video/:id/comment
```

**Request Body:**
```json
{
  "content": "This is a comment",
  "parent_id": "optional-parent-comment-id-for-replies"
}
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "id": "new-comment-id",
    "video_id": "video-uuid",
    "user_id": "user-uuid",
    "user_name": "User Display Name",
    "content": "This is a comment",
    "created_at": "timestamp",
    "updated_at": "timestamp",
    "likes": 0,
    "dislikes": 0,
    "status": "ACTIVE",
    "parent_id": "optional-parent-comment-id"
  }
}
```

### 4. Update a Comment

```
PUT /comment/:id
```

**Request Body:**
```json
{
  "content": "Updated comment content"
}
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "id": "comment-id",
    "content": "Updated comment content",
    "updated_at": "new-timestamp",
    "status": "ACTIVE"
  }
}
```

### 5. Delete a Comment (Soft Delete)

```
DELETE /comment/:id
```

**Response:**
```json
{
  "status": "success",
  "message": "Comment deleted successfully"
}
```

### 6. React to a Comment

```
POST /comment/:id/reaction
```

**Request Body:**
```json
{
  "type": "LIKE" // Enum: LIKE, DISLIKE
}
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "id": "comment-id",
    "likes": 11,
    "dislikes": 2
  }
}
```

### 7. Get Users Who Reacted to a Comment

```
GET /comment/:id/reactions?page=1&limit=20&type=LIKE
```

**Query Parameters:**
- `page`: Page number (default: 1)
- `limit`: Number of users per page (default: 20)
- `type`: Reaction type to filter by (options: "LIKE", "DISLIKE", default: all)

**Response:**
```json
{
  "status": "success",
  "data": {
    "users": [
      {
        "id": "user-uuid",
        "name": "User Display Name",
        "reaction_type": "LIKE",
        "created_at": "timestamp"
      }
    ],
    "total": 45,
    "page": 1,
    "limit": 20
  }
}
```

## Implementation Plan

### Phase 1: ScyllaDB Integration

1. Create a new ScyllaDB service in `internal/database/scylladb/`
2. Add ScyllaDB configuration to the application config
3. Implement connection and basic CRUD operations for ScyllaDB

### Phase 2: Redis Integration for Reaction Caching

1. Extend the Redis cache service with reaction-specific methods
2. Implement TTL-based caching for reaction data
3. Create background jobs for syncing Redis with ScyllaDB

### Phase 3: Pulsar Integration for Async Processing

1. Set up Pulsar topics for comment events
2. Implement Pulsar producer for reaction events
3. Create Pulsar consumers for batch processing

### Phase 4: Comments Package

1. Create a new comments package in `internal/comments/`
2. Implement the comments service with the following components:
   - `interface.go`: Define interfaces for the comments service
   - `model.go`: Define the comment model with proper enum types
   - `service.go`: Implement the business logic for comments
   - `handler.go`: Implement the HTTP handlers for comments
   - `types.go`: Define types for comments including enum types

### Phase 5: API Integration

1. Add comment routes to the application
2. Update the video service to include comment counts
3. Implement authentication and authorization for comment operations

### Phase 6: Testing

1. Write unit tests for the comments service
2. Write integration tests for the comments API
3. Perform load testing to ensure the system can handle high volumes of reactions

## Performance Optimizations

1. **Read Path Optimization:**
   - Cache hot comments' reaction data in Redis
   - Use materialized views for common query patterns
   - Implement pagination for all list endpoints

2. **Write Path Optimization:**
   - Use ScyllaDB's lightweight transactions for atomic operations
   - Batch write reactions for efficiency
   - Use Redis as a write-through cache for immediate user feedback

3. **Query Optimization:**
   - Partition keys designed for efficient comment retrieval by video
   - Clustering keys optimized for time-based sorting
   - Secondary indexes for user-based queries

## Security Considerations

1. **Authentication**: All comment operations except reading will require authentication
2. **Authorization**: Users can only edit or delete their own comments
3. **Rate Limiting**: Implement rate limiting to prevent spam
4. **Content Validation**: Validate comment content to prevent malicious input
5. **Data Privacy**: Ensure user data is properly protected in ScyllaDB

## Frontend Considerations

1. **Comment UI**: Design a clean, intuitive interface for displaying and interacting with comments
2. **Real-time Updates**: Consider implementing WebSockets for real-time comment updates
3. **Pagination**: Implement infinite scrolling or pagination for comments
4. **Markdown Support**: Consider supporting basic markdown in comments
5. **Responsive Design**: Ensure the comment UI works well on all device sizes
6. **Optimistic Updates**: Update UI immediately before server confirmation for reactions

--- 