---
# Comment Feature Specification

## Overview

The Comment feature allows users to add, view, edit, and delete comments on videos in the Pavilion Network platform. Unlike other data in the system which is stored in CockroachDB, comments will be stored in ScyllaDB, a high-performance NoSQL database, to better handle the sequential and potentially high-volume nature of comment data.

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
  status text -- ENUM: 'ACTIVE', 'FLAGGED', 'HIDDEN'
);
```

This table stores the primary comment data. Each comment has a unique UUID as its primary key and includes fields for tracking the video it belongs to, the user who created it, the content, timestamps, parent comment (for replies), and engagement metrics.

2. **Comments By Video Table**

```cql
CREATE TABLE comments_by_video (
  video_id uuid,
  comment_id uuid,
  created_at timestamp,
  PRIMARY KEY (video_id, created_at, comment_id)
) WITH CLUSTERING ORDER BY (created_at DESC, comment_id ASC);
```

This table optimizes queries for retrieving comments for a specific video. It organizes comments by video ID and created timestamp, allowing for efficient pagination and time-based sorting. The clustering order ensures newest comments appear first in query results.

3. **Replies Table**

```cql
CREATE TABLE replies (
  parent_id uuid,
  comment_id uuid,
  created_at timestamp,
  PRIMARY KEY (parent_id, created_at, comment_id)
) WITH CLUSTERING ORDER BY (created_at DESC, comment_id ASC);
```

This table optimizes the retrieval of replies to a specific comment. It organizes replies by parent comment ID and creation time, allowing for efficient thread-based views. The clustering order ensures newest replies appear first in query results.

4. **Reactions Table**

```cql
CREATE TABLE reactions (
  comment_id uuid,
  user_id uuid,
  type text, -- ENUM: 'LIKE', 'DISLIKE'
  created_at timestamp,
  PRIMARY KEY (comment_id, user_id)
);
```

This table tracks user reactions to comments. The primary key combination of comment_id and user_id ensures that a user can only have one reaction per comment, preventing duplicate reactions. The type field indicates whether the reaction is a like or dislike.

### Table Relationships and Access Patterns

The ScyllaDB schema is optimized for the following common access patterns:

1. **Video-Comment Relationship:**
   - One video can have many comments
   - The `comments_by_video` table facilitates efficient retrieval of comments for a specific video
   - Pagination and time-based sorting are handled through the clustering order

2. **Comment Threading:**
   - Comments can have parent-child relationships (nested comments)
   - Parent comments are stored in the main `comments` table
   - Child comments (replies) are indexed in the `replies` table for efficient retrieval
   - This design enables fast loading of comment threads

3. **User Engagement:**
   - Users can react to comments with likes or dislikes
   - The `reactions` table tracks individual user reactions
   - The primary key structure prevents duplicate reactions from the same user
   - Comment like/dislike counts are maintained in the `comments` table

4. **Performance Optimizations:**
   - Denormalized tables (`comments_by_video`, `replies`) enable high-performance reads
   - Composite primary keys with clustering orders support efficient filtering and pagination
   - The schema is optimized for read-heavy workloads, common in comment systems

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

1. **Video Service**: The Comment service will interact with the Video service to verify video existence.
2. **Auth Service**: The Comment service will use the Auth service to verify user authentication and authorization.
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

### 7. Remove a Reaction from a Comment

```
DELETE /comment/:id/reaction
```

**Response:**
```json
{
  "status": "success",
  "data": {
    "id": "comment-id",
    "likes": 10,
    "dislikes": 2
  }
}
``` 