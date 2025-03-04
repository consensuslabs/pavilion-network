#!/bin/bash
set -e

# Wait for Pulsar to be ready
echo "Waiting for Pulsar to be ready..."
until bin/pulsar-admin brokers healthcheck; do
    echo "Pulsar is not ready yet - waiting 5 seconds..."
    sleep 5
done

echo "Creating Pulsar topics for notification system..."

# Create topics with proper configuration
bin/pulsar-admin namespaces create pavilion/notifications
bin/pulsar-admin namespaces set-retention pavilion/notifications --size 1024M --time 48h

# Create the required topics
bin/pulsar-admin topics create persistent://pavilion/notifications/video-events
bin/pulsar-admin topics create persistent://pavilion/notifications/comment-events
bin/pulsar-admin topics create persistent://pavilion/notifications/user-events
bin/pulsar-admin topics create persistent://pavilion/notifications/dead-letter
bin/pulsar-admin topics create persistent://pavilion/notifications/retry-queue

# Enable deduplication at namespace level
bin/pulsar-admin namespaces set-deduplication pavilion/notifications --enable

echo "Pulsar topics have been successfully created with proper retention and deduplication settings."
