package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// Config represents database configuration
type Config struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
	SSLMode  string `yaml:"ssl_mode"`
}

// DB wraps a SQL database connection
type DB struct {
	*sql.DB
	config Config
}

// NewDB creates a new database connection
func NewDB(config Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode)
	
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	db := &DB{
		DB:     sqlDB,
		config: config,
	}
	
	return db, nil
}

// Migrate runs database migrations
func (db *DB) Migrate() error {
	queries := []string{
		createFilesTable,
		createResourcesTable,
		createFilesIndices,
		createResourcesIndices,
	}
	
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}
	}
	
	return nil
}

const createFilesTable = `
CREATE TABLE IF NOT EXISTS terraform_files (
    id VARCHAR(255) PRIMARY KEY,
    source VARCHAR(50) NOT NULL,
    source_path TEXT NOT NULL,
    file_type VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    content_hash VARCHAR(255) NOT NULL,
    collected_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);`

const createResourcesTable = `
CREATE TABLE IF NOT EXISTS terraform_objects (
    id VARCHAR(255) PRIMARY KEY,
    file_id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(255),
    configuration JSONB,
    dependencies TEXT[],
    address VARCHAR(500) NOT NULL,
    mode VARCHAR(50),
    provider_name VARCHAR(255),
    schema_version INTEGER,
    sensitive_values TEXT[],
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (file_id) REFERENCES terraform_files(id) ON DELETE CASCADE
);`

const createFilesIndices = `
CREATE INDEX IF NOT EXISTS idx_terraform_files_source ON terraform_files(source);
CREATE INDEX IF NOT EXISTS idx_terraform_files_file_type ON terraform_files(file_type);
CREATE INDEX IF NOT EXISTS idx_terraform_files_content_hash ON terraform_files(content_hash);
CREATE INDEX IF NOT EXISTS idx_terraform_files_updated_at ON terraform_files(updated_at);`

const createResourcesIndices = `
CREATE INDEX IF NOT EXISTS idx_terraform_objects_file_id ON terraform_objects(file_id);
CREATE INDEX IF NOT EXISTS idx_terraform_objects_type ON terraform_objects(type);
CREATE INDEX IF NOT EXISTS idx_terraform_objects_resource_type ON terraform_objects(resource_type);
CREATE INDEX IF NOT EXISTS idx_terraform_objects_address ON terraform_objects(address);
CREATE INDEX IF NOT EXISTS idx_terraform_objects_provider_name ON terraform_objects(provider_name);
CREATE INDEX IF NOT EXISTS idx_terraform_objects_updated_at ON terraform_objects(updated_at);`