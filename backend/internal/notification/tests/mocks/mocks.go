package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/consensuslabs/pavilion-network/backend/internal/logger"
	"github.com/consensuslabs/pavilion-network/backend/internal/notification/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

//
// Pulsar Message Mocks
//

// MockPulsarMessage implements a mock Pulsar message for testing
type MockPulsarMessage struct {
	payload    []byte
	key        string
	properties map[string]string
	acked      bool
	nacked     bool
}

// NewMockMessage creates a new mock message
func NewMockMessage(payload []byte, key string) *MockPulsarMessage {
	return &MockPulsarMessage{
		payload:    payload,
		key:        key,
		properties: make(map[string]string),
	}
}

// Payload returns the message payload
func (m *MockPulsarMessage) Payload() []byte {
	return m.payload
}

// ID returns a message ID
func (m *MockPulsarMessage) ID() pulsar.MessageID {
	return nil // Return nil for simplicity
}

// Key returns the message key
func (m *MockPulsarMessage) Key() string {
	return m.key
}

// Properties returns the message properties
func (m *MockPulsarMessage) Properties() map[string]string {
	return m.properties
}

// PublishTime returns the publish time
func (m *MockPulsarMessage) PublishTime() time.Time {
	return time.Now()
}

// EventTime returns the event time
func (m *MockPulsarMessage) EventTime() time.Time {
	return time.Now()
}

// BrokerPublishTime returns the broker publish time
func (m *MockPulsarMessage) BrokerPublishTime() *time.Time {
	t := time.Now()
	return &t
}

// Topic returns the topic name
func (m *MockPulsarMessage) Topic() string {
	return "mock-topic"
}

// ProducerName returns the producer name
func (m *MockPulsarMessage) ProducerName() string {
	return "mock-producer"
}

// RedeliveryCount returns the redelivery count
func (m *MockPulsarMessage) RedeliveryCount() uint32 {
	return 0
}

// SchemaVersion returns the schema version
func (m *MockPulsarMessage) SchemaVersion() []byte {
	return nil
}

// Index returns the index
func (m *MockPulsarMessage) Index() *uint64 {
	return nil
}

// OrderingKey returns the ordering key
func (m *MockPulsarMessage) OrderingKey() string {
	return ""
}

// GetReplicatedFrom returns replication info
func (m *MockPulsarMessage) GetReplicatedFrom() string {
	return ""
}

// GetSchemaValue gets a schema value
func (m *MockPulsarMessage) GetSchemaValue(v interface{}) error {
	return nil
}

// GetValue gets a value
func (m *MockPulsarMessage) GetValue(v interface{}) error {
	return nil
}

// GetEncryptionContext gets encryption context
func (m *MockPulsarMessage) GetEncryptionContext() *pulsar.EncryptionContext {
	return nil
}

// IsReplicated checks if message is replicated
func (m *MockPulsarMessage) IsReplicated() bool {
	return false
}

// SetProperty sets a property
func (m *MockPulsarMessage) SetProperty(key, value string) {
	m.properties[key] = value
}

// Ack acknowledges the message
func (m *MockPulsarMessage) Ack() error {
	m.acked = true
	return nil
}

// Nack rejects the message
func (m *MockPulsarMessage) Nack() error {
	m.nacked = true
	return nil
}

// IsAcked returns ack status
func (m *MockPulsarMessage) IsAcked() bool {
	return m.acked
}

// IsNacked returns nack status
func (m *MockPulsarMessage) IsNacked() bool {
	return m.nacked
}

//
// Logger Mock
//

// MockLogger is a mock implementation of logger.Logger
type MockLogger struct {
	mock.Mock
}

// LogInfo logs an info message
func (m *MockLogger) LogInfo(msg string, fields map[string]interface{}) {
	m.Called(msg, fields)
}

// LogError logs an error message
func (m *MockLogger) LogError(err error, msg string) error {
	args := m.Called(err, msg)
	return args.Error(0)
}

// LogErrorf logs a formatted error message
func (m *MockLogger) LogErrorf(err error, format string, args ...interface{}) error {
	mockArgs := m.Called(err, format, args)
	return mockArgs.Error(0)
}

// LogFatal logs a fatal message
func (m *MockLogger) LogFatal(err error, context string) {
	m.Called(err, context)
}

// LogDebug logs a debug message
func (m *MockLogger) LogDebug(message string, fields map[string]interface{}) {
	m.Called(message, fields)
}

// LogWarn logs a warning message
func (m *MockLogger) LogWarn(message string, fields map[string]interface{}) {
	m.Called(message, fields)
}

// WithFields adds fields to the logger
func (m *MockLogger) WithFields(fields map[string]interface{}) logger.Logger {
	args := m.Called(fields)
	return args.Get(0).(logger.Logger)
}

// WithContext adds context to the logger
func (m *MockLogger) WithContext(ctx context.Context) logger.Logger {
	args := m.Called(ctx)
	return args.Get(0).(logger.Logger)
}

// WithRequestID adds request ID to the logger
func (m *MockLogger) WithRequestID(requestID string) logger.Logger {
	args := m.Called(requestID)
	return args.Get(0).(logger.Logger)
}

// WithUserID adds user ID to the logger
func (m *MockLogger) WithUserID(userID string) logger.Logger {
	args := m.Called(userID)
	return args.Get(0).(logger.Logger)
}

//
// Repository Mock
//

// MockRepository is a simple in-memory repository for testing
type MockRepository struct {
	notifications map[uuid.UUID]*types.Notification
	unreadCount   map[uuid.UUID]int
	mutex         sync.RWMutex
	mock.Mock
}

// NewMockRepository creates a new mock repository
func NewMockRepository() *MockRepository {
	return &MockRepository{
		notifications: make(map[uuid.UUID]*types.Notification),
		unreadCount:   make(map[uuid.UUID]int),
	}
}

// SaveNotification saves a notification
func (r *MockRepository) SaveNotification(ctx context.Context, notification *types.Notification) error {
	args := r.Mock.Called(ctx, notification)
	if args.Get(0) != nil {
		return args.Error(0)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.notifications[notification.ID] = notification

	// Update unread count
	if notification.ReadAt == nil {
		r.unreadCount[notification.UserID]++
	}

	return nil
}

// GetNotificationsByUserID gets notifications for a user
func (r *MockRepository) GetNotificationsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*types.Notification, error) {
	args := r.Mock.Called(ctx, userID, limit, offset)
	if args.Get(0) != nil {
		return args.Get(0).([]*types.Notification), args.Error(1)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*types.Notification
	for _, n := range r.notifications {
		if n.UserID == userID {
			result = append(result, n)
		}
	}

	// Apply offset and limit
	if offset >= len(result) {
		return []*types.Notification{}, nil
	}

	end := offset + limit
	if end > len(result) {
		end = len(result)
	}

	return result[offset:end], nil
}

// GetUnreadCount gets the count of unread notifications
func (r *MockRepository) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	args := r.Mock.Called(ctx, userID)
	if args.Get(1) != nil {
		return args.Int(0), args.Error(1)
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.unreadCount[userID], nil
}

// MarkAsRead marks a notification as read
func (r *MockRepository) MarkAsRead(ctx context.Context, notificationID uuid.UUID) error {
	args := r.Mock.Called(ctx, notificationID)
	if args.Get(0) != nil {
		return args.Error(0)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	n, exists := r.notifications[notificationID]
	if !exists {
		return nil
	}

	// Check if it's already read
	if n.ReadAt != nil {
		return nil
	}

	// Mark as read
	now := time.Now()
	n.ReadAt = &now

	// Update unread count
	r.unreadCount[n.UserID]--
	if r.unreadCount[n.UserID] < 0 {
		r.unreadCount[n.UserID] = 0
	}

	return nil
}

// MarkAllAsRead marks all notifications for a user as read
func (r *MockRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	args := r.Mock.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Error(0)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	if ctx.Value("now") != nil {
		nowFunc := ctx.Value("now").(func() time.Time)
		now = nowFunc()
	}

	for _, n := range r.notifications {
		if n.UserID == userID && n.ReadAt == nil {
			n.ReadAt = &now
		}
	}

	r.unreadCount[userID] = 0

	return nil
}

// Close closes the repository connection
func (r *MockRepository) Close() error {
	args := r.Mock.Called()
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return nil
}

// Ping checks repository health
func (r *MockRepository) Ping(ctx context.Context) error {
	args := r.Mock.Called(ctx)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	return nil
}

// GetNotifications returns all notifications in the repository
func (r *MockRepository) GetNotifications() []*types.Notification {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	result := make([]*types.Notification, 0, len(r.notifications))
	for _, n := range r.notifications {
		result = append(result, n)
	}
	return result
}

//
// Pulsar Client Mock
//

// MockPulsarClient is a mock implementation of pulsar.Client
type MockPulsarClient struct {
	mock.Mock
}

// CreateProducer creates a mock producer
func (m *MockPulsarClient) CreateProducer(options pulsar.ProducerOptions) (pulsar.Producer, error) {
	args := m.Called(options)
	return args.Get(0).(pulsar.Producer), args.Error(1)
}

// CreateConsumer creates a mock consumer
func (m *MockPulsarClient) CreateConsumer(options pulsar.ConsumerOptions) (pulsar.Consumer, error) {
	args := m.Called(options)
	return args.Get(0).(pulsar.Consumer), args.Error(1)
}

// Subscribe subscribes to a topic
func (m *MockPulsarClient) Subscribe(options pulsar.ConsumerOptions) (pulsar.Consumer, error) {
	args := m.Called(options)
	return args.Get(0).(pulsar.Consumer), args.Error(1)
}

// CreateReader creates a mock reader
func (m *MockPulsarClient) CreateReader(options pulsar.ReaderOptions) (pulsar.Reader, error) {
	args := m.Called(options)
	return args.Get(0).(pulsar.Reader), args.Error(1)
}

// TopicPartitions returns topic partitions
func (m *MockPulsarClient) TopicPartitions(topic string) ([]string, error) {
	args := m.Called(topic)
	return args.Get(0).([]string), args.Error(1)
}

// Close closes the client
func (m *MockPulsarClient) Close() {
	m.Called()
}

// CreateTableView creates a mock table view
func (m *MockPulsarClient) CreateTableView(options pulsar.TableViewOptions) (pulsar.TableView, error) {
	args := m.Called(options)
	return args.Get(0).(pulsar.TableView), args.Error(1)
}

// NewTransaction creates a mock transaction
func (m *MockPulsarClient) NewTransaction(timeout time.Duration) (pulsar.Transaction, error) {
	args := m.Called(timeout)
	return args.Get(0).(pulsar.Transaction), args.Error(1)
}

//
// Pulsar Producer Mock
//

// MockPulsarProducer is a mock implementation of pulsar.Producer
type MockPulsarProducer struct {
	mock.Mock
	sentMessages []pulsar.ProducerMessage
}

// Send sends a message
func (m *MockPulsarProducer) Send(ctx context.Context, msg *pulsar.ProducerMessage) (pulsar.MessageID, error) {
	args := m.Called(ctx, msg)
	m.sentMessages = append(m.sentMessages, *msg)
	return args.Get(0).(pulsar.MessageID), args.Error(1)
}

// SendAsync sends a message asynchronously
func (m *MockPulsarProducer) SendAsync(ctx context.Context, msg *pulsar.ProducerMessage, callback func(pulsar.MessageID, *pulsar.ProducerMessage, error)) {
	m.Called(ctx, msg, callback)
}

// LastSequenceID returns the last sequence ID
func (m *MockPulsarProducer) LastSequenceID() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// Flush flushes the producer
func (m *MockPulsarProducer) Flush() error {
	args := m.Called()
	return args.Error(0)
}

// FlushWithCtx flushes the producer with context
func (m *MockPulsarProducer) FlushWithCtx(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Topic returns the topic name
func (m *MockPulsarProducer) Topic() string {
	args := m.Called()
	return args.String(0)
}

// Name returns the producer name
func (m *MockPulsarProducer) Name() string {
	args := m.Called()
	return args.String(0)
}

// Close closes the producer
func (m *MockPulsarProducer) Close() {
	m.Called()
}

// GetSentMessages returns sent messages
func (m *MockPulsarProducer) GetSentMessages() []pulsar.ProducerMessage {
	return m.sentMessages
}

//
// Pulsar Consumer Mock
//

// MockPulsarConsumer is a mock implementation of pulsar.Consumer
type MockPulsarConsumer struct {
	mock.Mock
}

// Subscription returns the subscription name
func (m *MockPulsarConsumer) Subscription() string {
	args := m.Called()
	return args.String(0)
}

// Unsubscribe unsubscribes from the topic
func (m *MockPulsarConsumer) Unsubscribe() error {
	args := m.Called()
	return args.Error(0)
}

// Receive receives a message
func (m *MockPulsarConsumer) Receive(ctx context.Context) (pulsar.Message, error) {
	args := m.Called(ctx)
	return args.Get(0).(pulsar.Message), args.Error(1)
}

// Chan returns a channel for messages
func (m *MockPulsarConsumer) Chan() <-chan pulsar.ConsumerMessage {
	args := m.Called()
	return args.Get(0).(<-chan pulsar.ConsumerMessage)
}

// Ack acknowledges a message
func (m *MockPulsarConsumer) Ack(msg pulsar.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

// AckID acknowledges a message by ID
func (m *MockPulsarConsumer) AckID(msgID pulsar.MessageID) error {
	args := m.Called(msgID)
	return args.Error(0)
}

// Nack rejects a message
func (m *MockPulsarConsumer) Nack(msg pulsar.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

// NackID rejects a message by ID
func (m *MockPulsarConsumer) NackID(msgID pulsar.MessageID) error {
	args := m.Called(msgID)
	return args.Error(0)
}

// Close closes the consumer
func (m *MockPulsarConsumer) Close() {
	m.Called()
}

// Seek seeks to a position
func (m *MockPulsarConsumer) Seek(msgID pulsar.MessageID) error {
	args := m.Called(msgID)
	return args.Error(0)
}

// SeekByTime seeks to a time
func (m *MockPulsarConsumer) SeekByTime(time time.Time) error {
	args := m.Called(time)
	return args.Error(0)
}

//
// Test Helpers
//

// SetupTestLogger creates and configures a mock logger for tests
func SetupTestLogger() *MockLogger {
	mockLogger := new(MockLogger)
	mockLogger.On("LogInfo", mock.Anything, mock.Anything).Return()
	mockLogger.On("LogError", mock.Anything, mock.Anything).Return(nil)
	mockLogger.On("LogDebug", mock.Anything, mock.Anything).Return()
	mockLogger.On("LogWarn", mock.Anything, mock.Anything).Return()
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	return mockLogger
}

// SetupTestRepository creates a mock repository for tests
func SetupTestRepository() *MockRepository {
	repo := NewMockRepository()
	repo.On("SaveNotification", mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNotificationsByUserID", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	repo.On("GetUnreadCount", mock.Anything, mock.Anything).Return(0, nil)
	repo.On("MarkAsRead", mock.Anything, mock.Anything).Return(nil)
	repo.On("MarkAllAsRead", mock.Anything, mock.Anything).Return(nil)
	repo.On("Close").Return(nil)
	repo.On("Ping", mock.Anything).Return(nil)
	return repo
}

// SetupConsumerTest creates and configures common mocks for consumer tests
func SetupConsumerTest() (*MockLogger, *MockRepository) {
	return SetupTestLogger(), SetupTestRepository()
}

// setupConsumerTest is an alias for SetupConsumerTest to maintain backward compatibility
func setupConsumerTest() (*MockLogger, *MockRepository) {
	return SetupConsumerTest()
}

// setupTestLogger is an alias for SetupTestLogger to maintain backward compatibility
func setupTestLogger() *MockLogger {
	return SetupTestLogger()
}

// setupTestRepository is an alias for SetupTestRepository to maintain backward compatibility
func setupTestRepository() *MockRepository {
	return SetupTestRepository()
}

//
// Additional Mock Types for Producer Testing
//

// MockMessageID is a mock implementation of pulsar.MessageID
type MockMessageID struct {
	mock.Mock
}

// Serialize serializes the message ID
func (m *MockMessageID) Serialize() []byte {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]byte)
}

// String returns a string representation of the message ID
func (m *MockMessageID) String() string {
	args := m.Called()
	return args.String(0)
}

// BatchIdx returns the batch index
func (m *MockMessageID) BatchIdx() int32 {
	args := m.Called()
	return args.Get(0).(int32)
}

// BatchSize returns the batch size
func (m *MockMessageID) BatchSize() int32 {
	args := m.Called()
	return args.Get(0).(int32)
}

// LedgerID returns the ledger ID
func (m *MockMessageID) LedgerID() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// EntryID returns the entry ID
func (m *MockMessageID) EntryID() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// PartitionIdx returns the partition index
func (m *MockMessageID) PartitionIdx() int32 {
	args := m.Called()
	return args.Get(0).(int32)
}

// MockTableView is a mock implementation of pulsar.TableView
type MockTableView struct {
	mock.Mock
}

// Get gets a value from the table view
func (m *MockTableView) Get(key string) []byte {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]byte)
}

// Size returns the size of the table view
func (m *MockTableView) Size() int {
	args := m.Called()
	return args.Int(0)
}

// ForEach iterates over the table view
func (m *MockTableView) ForEach(fn func(string, []byte)) {
	m.Called(fn)
}

// Range iterates over a range of the table view
func (m *MockTableView) Range(start, end string, fn func(string, []byte)) {
	m.Called(start, end, fn)
}

// Close closes the table view
func (m *MockTableView) Close() {
	m.Called()
}

// MockTransaction is a mock implementation of pulsar.Transaction
type MockTransaction struct {
	mock.Mock
}

// GetTxnID returns the transaction ID
func (m *MockTransaction) GetTxnID() pulsar.TxnID {
	args := m.Called()
	return args.Get(0).(pulsar.TxnID)
}

// Commit commits the transaction
func (m *MockTransaction) Commit() error {
	args := m.Called()
	return args.Error(0)
}

// Abort aborts the transaction
func (m *MockTransaction) Abort() error {
	args := m.Called()
	return args.Error(0)
}

// MockTxnID is a mock implementation of pulsar.TxnID
type MockTxnID struct {
	mock.Mock
}

// MostSigBits returns the most significant bits
func (m *MockTxnID) MostSigBits() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// LeastSigBits returns the least significant bits
func (m *MockTxnID) LeastSigBits() int64 {
	args := m.Called()
	return args.Get(0).(int64)
}

// SetupProducerTest creates and configures common mocks for producer tests
func SetupProducerTest() (*MockPulsarClient, *MockPulsarProducer, *MockLogger, *MockRepository) {
	mockClient := new(MockPulsarClient)
	mockProducer := new(MockPulsarProducer)
	mockLogger := SetupTestLogger()
	mockRepo := SetupTestRepository()
	
	// Create and configure mockMessageID
	mockMessageID := new(MockMessageID)
	mockMessageID.On("Serialize").Return([]byte("mock-id"))
	mockMessageID.On("String").Return("mock-id")
	mockMessageID.On("BatchIdx").Return(int32(0))
	mockMessageID.On("BatchSize").Return(int32(1))
	mockMessageID.On("LedgerID").Return(int64(1))
	mockMessageID.On("EntryID").Return(int64(1))
	mockMessageID.On("PartitionIdx").Return(int32(0))
	
	// Configure mocks
	mockClient.On("CreateProducer", mock.Anything).Return(mockProducer, nil)
	mockProducer.On("Send", mock.Anything, mock.Anything).Return(mockMessageID, nil)
	mockProducer.On("Close").Return()
	
	return mockClient, mockProducer, mockLogger, mockRepo
}