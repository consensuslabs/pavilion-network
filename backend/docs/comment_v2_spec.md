# Comment System V2 Specification

## Overview
This document outlines the technical specification for version 2 of the Pavilion Network comment system. The key focus is on optimizing for high-scale scenarios with thousands of simultaneous comments and reactions while maintaining system performance and stability.

## Current Architecture
- **Video Storage**: CockroachDB (relational DB)
- **Comments/Reactions**: ScyllaDB (NoSQL)
- **Current Tables**:
  - `comments`: Main comment data with likes/dislikes counters directly in the table
  - `comments_by_video`: Index table for retrieving comments by video
  - `replies`: Index table for retrieving replies to comments
  - `reactions`: Raw reaction data from users

## Optimization Approaches (Priority Order)

### 1. Eventual Consistency with Background Reconciliation
**Priority**: High (P0)
**Problem**: Exact, real-time counts for viral content create excessive write load and contention

**Solution**:
- Separate raw events (user actions) from computed metrics (counts)
- Use background processes to calculate and update aggregated metrics
- Accept temporary staleness in displayed counts for reduced system load

### 2. Redis Integration for Caching and Rate Limiting
**Priority**: High (P0)
**Problem**: Repeated reads and high-frequency writes to ScyllaDB

**Solution**:
- Implement a Redis caching layer for hot content
- Use Redis for fast counter updates during viral periods
- Add rate limiting to prevent abuse and control write throughput

### 3. Event Streaming with Apache Pulsar
**Priority**: Medium (P1)
**Problem**: Synchronous processing creates bottlenecks and increases latency

**Solution**:
- Implement an event-driven architecture using Apache Pulsar
- Process comments and reactions asynchronously
- Enable seamless scaling for processing components

### 4. ScyllaDB Schema Optimizations
**Priority**: Medium (P1)
**Problem**: Current schema not optimized for high-volume write patterns

**Solution**:
- Optimize table design for write distribution
- Implement time-bucketing for high-volume videos
- Use lightweight transactions (LWT) only when necessary

### 5. Read/Write Split Pattern
**Priority**: Medium (P1)
**Problem**: Mixing read and write operations creates contention

**Solution**:
- Separate read operations from write operations
- Serve reads primarily from cache or pre-computed views
- Focus write operations on event recording, not computation

### 6. Batching Operations
**Priority**: Low (P2)
**Problem**: Individual write operations create network overhead

**Solution**:
- Batch related write operations together
- Accumulate updates and persist them in groups
- Use batch imports for recreation of metrics

### 7. Optimistic Concurrency for Reactions
**Priority**: Low (P2)
**Problem**: Traditional locking creates bottlenecks for reaction updates

**Solution**:
- Implement optimistic concurrency control for reactions
- Accept and resolve potential conflicts after the fact
- Use idempotent operations where possible

## Phased Implementation Approach

The implementation will follow a four-phase approach spanning approximately 8 weeks:

1. **Foundation Phase** (Weeks 1-2): Core infrastructure for eventual consistency, Redis integration, and rate limiting
2. **Event Processing Phase** (Weeks 3-4): Event streaming infrastructure with Apache Pulsar
3. **Advanced Optimizations Phase** (Weeks 5-6): Schema optimizations and batching operations
4. **Scaling and Finalization Phase** (Weeks 7-8): Concurrent operations, monitoring, and performance testing

*For detailed implementation tasks, timelines, and technical specifics, see the companion document: `comment_v2_impl_plan.md`*

## Expected Benefits

### Performance Improvements
- Reduce peak database load by 80-90%
- Support 10x current comment and reaction volumes
- Maintain sub-100ms API response times at scale
- Support viral videos with millions of interactions

### User Experience Improvements
- Consistent response times even during viral events
- Reduced likelihood of rate limiting or errors during high traffic
- More accurate trending content identification

### Operational Improvements
- Better visibility into system performance with advanced monitoring
- Reduced operational burden during traffic spikes
- More graceful degradation under extreme load

## Success Criteria

### Functional Requirements
- All existing comment and reaction functionality must continue to work
- Comment counts and reactions must eventually be consistent with actual user actions
- The system must gracefully handle viral content with minimal degradation

### Non-Functional Requirements
- API endpoints must maintain response times under 100ms at p95 percentile
- The system must support at least 1,000 concurrent users per video
- Viral videos must support at least 100 comments/second and 1,000 reactions/second

## Integration Points
- Frontend UI may need updates to display approximate counts and handle eventual consistency
- Analytics systems should be updated to work with the new data structure
- Monitoring systems will need configuration for new metrics and alerts

## Conclusion
The Comment System V2 represents a significant architectural improvement that will allow the Pavilion Network platform to scale efficiently for viral content while maintaining performance and reliability. By implementing these optimizations in a phased approach, we can progressively enhance the system while minimizing disruption to users. 