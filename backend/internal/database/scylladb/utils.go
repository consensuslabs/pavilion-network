package scylladb

import (
	"encoding/json"
)

// encodeToJSONBytes serializes a value to JSON bytes
func encodeToJSONBytes(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// decodeFromJSONBytes deserializes JSON bytes to a value
func decodeFromJSONBytes(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}