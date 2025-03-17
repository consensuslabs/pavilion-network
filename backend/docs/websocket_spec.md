# WebSocket Specification

## Overview

This document outlines the WebSocket communication protocol for real-time notifications and updates in the Pavilion Network platform. WebSockets provide bidirectional communication channels between clients and the server, enabling instant delivery of notifications, status updates, and other real-time information.

## Connection

### Connection Endpoint

```
wss://api.pavilion.network/ws
```

### Authentication

WebSocket connections require authentication using a JWT token. The token should be passed as a query parameter:

```
wss://api.pavilion.network/ws?token=<JWT_TOKEN>
```

## Message Format

All WebSocket messages follow a standard JSON format:

```json
{
  "type": "string",      // Message type
  "payload": {},         // Message payload (varies by type)
  "timestamp": "string"  // ISO 8601 timestamp
}
```

## Message Types

### Server-to-Client Messages

#### `notification`

Sent when a new notification is created for the user.

```json
{
  "type": "notification",
  "payload": {
    "id": "string",
    "type": "string",
    "content": "string",
    "isRead": false,
    "createdAt": "string",
    "sender": {
      "id": "string",
      "username": "string",
      "profilePhoto": "string"
    },
    "reference": {
      "type": "string",
      "id": "string"
    }
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

#### `status_update`

Sent when a user's status changes (online, offline, busy, etc.).

```json
{
  "type": "status_update",
  "payload": {
    "userId": "string",
    "status": "string",
    "lastActive": "string"
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

#### `chat_message`

Sent when a new chat message is received.

```json
{
  "type": "chat_message",
  "payload": {
    "id": "string",
    "roomId": "string",
    "sender": {
      "id": "string",
      "username": "string",
      "profilePhoto": "string"
    },
    "content": "string",
    "createdAt": "string",
    "attachments": []
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

### Client-to-Server Messages

#### `subscribe`

Subscribe to a specific channel or topic.

```json
{
  "type": "subscribe",
  "payload": {
    "channel": "string",
    "parameters": {}
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

#### `unsubscribe`

Unsubscribe from a specific channel or topic.

```json
{
  "type": "unsubscribe",
  "payload": {
    "channel": "string"
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

#### `mark_read`

Mark a notification as read.

```json
{
  "type": "mark_read",
  "payload": {
    "notificationId": "string"
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

#### `presence`

Update user's presence status.

```json
{
  "type": "presence",
  "payload": {
    "status": "string"  // "online", "offline", "busy", "away"
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

## Heartbeat and Connection Maintenance

The server sends a heartbeat message every 30 seconds to keep the connection alive:

```json
{
  "type": "heartbeat",
  "timestamp": "2024-03-15T12:00:00Z"
}
```

Clients should respond with a heartbeat acknowledgment:

```json
{
  "type": "heartbeat_ack",
  "timestamp": "2024-03-15T12:00:00Z"
}
```

If the server does not receive any messages from a client for 90 seconds, the connection will be closed.

## Subscription Channels

### Available Channels

- `user`: User-specific notifications
- `zone:{zoneId}`: Zone-specific updates
- `channel:{channelId}`: Channel-specific updates
- `chat:{roomId}`: Chat room messages

### Example Subscriptions

```json
{
  "type": "subscribe",
  "payload": {
    "channel": "zone:123"
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

```json
{
  "type": "subscribe",
  "payload": {
    "channel": "chat:456"
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

## Error Handling

When an error occurs, the server sends an error message:

```json
{
  "type": "error",
  "payload": {
    "code": "number",
    "message": "string"
  },
  "timestamp": "2024-03-15T12:00:00Z"
}
```

### Error Codes

- `1001`: Authentication error
- `1002`: Invalid message format
- `1003`: Invalid subscription
- `1004`: Permission denied
- `1005`: Rate limit exceeded
- `2000`: Internal server error

## Implementation Guidelines

1. Clients should implement reconnection logic with exponential backoff
2. Clients should maintain a queue of unsent messages during disconnection
3. Handle connection errors gracefully and provide user feedback
4. Implement proper error handling for malformed messages

## Security Considerations

1. All WebSocket connections must use WSS (WebSocket Secure)
2. JWT tokens should be short-lived and renewed regularly
3. Validate all incoming messages to prevent injection attacks
4. Implement rate limiting to prevent abuse
5. Apply proper access control on subscription channels