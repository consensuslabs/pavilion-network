package testhelper

import (
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/gocql/gocql"
)

// SetupNotificationTestConfig loads and returns configuration for notification tests
func SetupNotificationTestConfig() (*Config, error) {
	// Load the full test configuration
	config, err := LoadTestConfig()
	if err != nil {
		return nil, err
	}
	
	// If ScyllaDB hosts is empty, set default
	if len(config.ScyllaDB.Hosts) == 0 {
		config.ScyllaDB.Hosts = []string{"localhost:9042"}
	}
	
	// If ScyllaDB keyspace is empty, set default
	if config.ScyllaDB.Keyspace == "" {
		config.ScyllaDB.Keyspace = "pavilion_test" // Use the keyspace from updated config_test.yaml
	}
	
	// If ScyllaDB consistency is empty, set default
	if config.ScyllaDB.Consistency == "" {
		config.ScyllaDB.Consistency = "one"
	}
	
	// If ScyllaDB timeout is not set, set default
	if config.ScyllaDB.Timeout == "" {
		config.ScyllaDB.Timeout = "30s"
	}
	
	// If Pulsar URL is empty, set default
	if config.Pulsar.URL == "" {
		config.Pulsar.URL = "pulsar://localhost:6650"
	}
	
	// If Pulsar operation timeout is not set, set default
	if config.Pulsar.OperationTimeout == 0 {
		config.Pulsar.OperationTimeout = 30
	}
	
	// If Pulsar connection timeout is not set, set default
	if config.Pulsar.ConnectionTimeout == 0 {
		config.Pulsar.ConnectionTimeout = 30
	}
	
	return config, nil
}

// CreatePulsarClient creates a real Pulsar client for testing
func CreatePulsarClient(config *Config) (pulsar.Client, error) {
	return pulsar.NewClient(pulsar.ClientOptions{
		URL:               config.Pulsar.URL,
		OperationTimeout:  time.Duration(config.Pulsar.OperationTimeout) * time.Second,
		ConnectionTimeout: time.Duration(config.Pulsar.ConnectionTimeout) * time.Second,
	})
}

// CreateScyllaDBSession creates a real ScyllaDB session for testing
func CreateScyllaDBSession(config *Config) (*gocql.Session, error) {
	// Parse the timeout
	var timeout time.Duration
	if config.ScyllaDB.Timeout != "" {
		var err error
		timeout, err = time.ParseDuration(config.ScyllaDB.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid ScyllaDB timeout format: %w", err)
		}
	} else {
		timeout = 30 * time.Second // Default timeout
	}
	
	// First, try to connect without a keyspace to create it if needed
	systemCluster := gocql.NewCluster(config.ScyllaDB.Hosts...)
	systemCluster.Keyspace = "system"
	systemCluster.Consistency = gocql.ParseConsistency(config.ScyllaDB.Consistency)
	systemCluster.Timeout = timeout
	
	systemSession, err := systemCluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ScyllaDB system keyspace: %w", err)
	}
	defer systemSession.Close()
	
	// Check if keyspace exists
	var count int
	if err := systemSession.Query(`SELECT count(*) FROM system_schema.keyspaces WHERE keyspace_name = ?`, 
		config.ScyllaDB.Keyspace).Scan(&count); err != nil {
		return nil, fmt.Errorf("failed to check if keyspace exists: %w", err)
	}
	
	// Create keyspace if it doesn't exist
	if count == 0 {
		fmt.Printf("Creating keyspace %s...\n", config.ScyllaDB.Keyspace)
		createKeyspace := fmt.Sprintf(`
			CREATE KEYSPACE IF NOT EXISTS %s 
			WITH REPLICATION = { 
				'class' : 'SimpleStrategy', 
				'replication_factor' : 1 
			}`, config.ScyllaDB.Keyspace)
		
		if err := systemSession.Query(createKeyspace).Exec(); err != nil {
			return nil, fmt.Errorf("failed to create keyspace: %w", err)
		}
	}
	
	// Now, connect to the keyspace
	cluster := gocql.NewCluster(config.ScyllaDB.Hosts...)
	cluster.Keyspace = config.ScyllaDB.Keyspace
	cluster.Consistency = gocql.ParseConsistency(config.ScyllaDB.Consistency)
	cluster.Timeout = timeout
	
	// Connect to ScyllaDB with the desired keyspace
	return cluster.CreateSession()
}

// CheckPulsarAvailable verifies if Pulsar is accessible
func CheckPulsarAvailable(config *Config) bool {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL:               config.Pulsar.URL,
		OperationTimeout:  5 * time.Second, // Short timeout for checking
		ConnectionTimeout: 5 * time.Second,
	})
	if err != nil {
		return false
	}
	client.Close()
	return true
}

// CheckScyllaDBAvailable verifies if ScyllaDB is accessible
func CheckScyllaDBAvailable(config *Config) bool {
	cluster := gocql.NewCluster(config.ScyllaDB.Hosts...)
	cluster.Keyspace = "system"
	cluster.Timeout = 5 * time.Second
	
	session, err := cluster.CreateSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
} 