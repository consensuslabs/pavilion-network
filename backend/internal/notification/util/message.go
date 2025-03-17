package util

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/google/uuid"
)

// GenerateEventID generates a new event ID if not provided
func GenerateEventID(id uuid.UUID) uuid.UUID {
	if id == uuid.Nil {
		return uuid.New()
	}
	return id
}

// GenerateEventTime generates a new event time if not provided
func GenerateEventTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

// GenerateSequenceNumber generates a new sequence number if not provided
func GenerateSequenceNumber(seq int64) int64 {
	if seq == 0 {
		return time.Now().UnixNano()
	}
	return seq
}

// CreateProducerMessage creates a producer message from an event
func CreateProducerMessage(event interface{}, key string, properties map[string]string, eventTime time.Time) (*pulsar.ProducerMessage, error) {
	// Serialize the event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event: %w", err)
	}

	// Create producer message
	msg := &pulsar.ProducerMessage{
		Payload:    data,
		Key:        key,
		Properties: properties,
		EventTime:  eventTime,
	}

	return msg, nil
}

// TruncateContent truncates a string to the specified length and adds ellipsis if needed
func TruncateContent(content string, maxLength int) string {
	if len(content) <= maxLength {
		return content
	}
	return content[:maxLength-3] + "..."
} 