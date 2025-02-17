package database

import "time"

// MigrationRecord tracks which migrations have been executed
type MigrationRecord struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"not null"` // Migration file name
	Hash      string    `gorm:"not null"` // Hash of migration content for integrity
	AppliedAt time.Time `gorm:"not null"`
	BatchNo   int       `gorm:"not null"` // Batch number for grouping migrations
}

// TableName specifies the table name for migration records
func (MigrationRecord) TableName() string {
	return "schema_migrations"
}
