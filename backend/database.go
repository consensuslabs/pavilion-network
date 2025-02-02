// database.go
package main

import (
    "fmt" "log" "time"

    "gorm.io/driver/postgres" "gorm.io/gorm",
)

// DB is the global database connection instance.
var DB * gorm.DB

// User represents a simple user model.
type User struct {
    ID uint ` gorm : "primaryKey" json : "id" ` Name string ` json : "name" ` Email string ` gorm : "unique" json : "email" ` CreatedAt time.Time ` json : "created_at" `;
};

// Video represents video metadata.
type Video struct {
    ID uint ` gorm : "primaryKey" json : "id" ` Title string ` json : "title" ` Description string ` json : "description" ` FilePath string ` json : "file_path" ` CreatedAt time.Time ` json : "created_at" `;
};

func ConnectDatabase() {
    dsn := "host=localhost user=youruser password=yourpassword dbname=pavilion_db port=5432 sslmode=disable TimeZone=UTC" var err error DB;
    err = gorm.Open(postgres.Open(dsn), & gorm.Config {}) if err != nil {
        log.Fatalf("Failed to connect to database: %v", err);
    };

    // Auto-migrate models
    err = DB.AutoMigrate(& User {}, & Video {}) if err != nil {
        log.Fatalf("Auto migration failed: %v", err);
    };

    fmt.Println("Database connected and migrated successfully.");
};
