package scylladb

import (
	"fmt"

	"github.com/consensuslabs/pavilion-network/backend/internal/video"
	"github.com/gocql/gocql"
)

// Client manages connections to ScyllaDB
type Client struct {
	config  Config
	session *gocql.Session
	logger  video.Logger
	schema  *SchemaManager
}

// NewClient creates a new ScyllaDB client
func NewClient(config Config, logger video.Logger) *Client {
	return &Client{
		config: config,
		logger: logger,
	}
}

// Connect establishes a connection to the ScyllaDB cluster
func (c *Client) Connect() error {
	cluster := gocql.NewCluster(c.config.Hosts...)
	cluster.Port = c.config.Port
	cluster.Consistency = getConsistencyLevel(c.config.Consistency)

	if c.config.Username != "" && c.config.Password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: c.config.Username,
			Password: c.config.Password,
		}
	}

	// Set timeouts
	cluster.Timeout = c.config.Timeout
	cluster.ConnectTimeout = c.config.ConnectTimeout

	// Connect without keyspace initially
	var err error
	c.session, err = cluster.CreateSession()
	if err != nil {
		c.logger.LogError("Failed to connect to ScyllaDB", map[string]interface{}{"error": err.Error()})
		return err
	}

	// Create schema manager
	c.schema = NewSchemaManager(c.session, c.config, c.logger)

	// Create keyspace if it doesn't exist
	if err := c.schema.CreateKeyspaceIfNotExists(); err != nil {
		c.logger.LogError("Failed to create keyspace", map[string]interface{}{"error": err.Error()})
		return err
	}

	// Close the initial session
	c.session.Close()

	// Reconnect with the keyspace
	cluster.Keyspace = c.config.Keyspace
	c.session, err = cluster.CreateSession()
	if err != nil {
		c.logger.LogError("Failed to connect to ScyllaDB with keyspace", map[string]interface{}{
			"error":    err.Error(),
			"keyspace": c.config.Keyspace,
		})
		return err
	}

	c.logger.LogInfo("Connected to ScyllaDB", map[string]interface{}{
		"hosts":    c.config.Hosts,
		"keyspace": c.config.Keyspace,
	})

	// Update schema manager with new session
	c.schema = NewSchemaManager(c.session, c.config, c.logger)

	// Initialize schema
	if err := c.schema.InitializeSchema(); err != nil {
		c.logger.LogError("Failed to initialize schema", map[string]interface{}{"error": err.Error()})
		return err
	}

	return nil
}

// Close closes the connection to the ScyllaDB cluster
func (c *Client) Close() error {
	if c.session != nil {
		c.session.Close()
		c.logger.LogInfo("Closed connection to ScyllaDB", nil)
	}
	return nil
}

// Session returns the current database session
func (c *Client) Session() *gocql.Session {
	return c.session
}

// Ping checks if the connection is alive
func (c *Client) Ping() error {
	if c.session == nil {
		return fmt.Errorf("session is not established")
	}

	// Simple query to verify connection
	var version string
	if err := c.session.Query("SELECT release_version FROM system.local").Scan(&version); err != nil {
		return err
	}

	return nil
}

// getConsistencyLevel converts string consistency level to gocql.Consistency
func getConsistencyLevel(level string) gocql.Consistency {
	switch level {
	case "one":
		return gocql.One
	case "quorum":
		return gocql.Quorum
	case "all":
		return gocql.All
	case "local_quorum":
		return gocql.LocalQuorum
	case "each_quorum":
		return gocql.EachQuorum
	default:
		return gocql.Quorum // Default to Quorum
	}
}
