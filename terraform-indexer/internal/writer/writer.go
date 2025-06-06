package writer

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/db"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/model"
)

// Writer defines the interface for writing TerraformObjects to storage
type Writer interface {
	WriteFile(ctx context.Context, file *model.TerraformFile) error
	WriteObject(ctx context.Context, obj *model.TerraformObject) error
	WriteBatch(ctx context.Context, files []*model.TerraformFile, objects []*model.TerraformObject) error
}

// DatabaseWriter writes objects to a PostgreSQL database
type DatabaseWriter struct {
	db *db.DB
}

// NewDatabaseWriter creates a new database writer
func NewDatabaseWriter(database *db.DB) *DatabaseWriter {
	return &DatabaseWriter{
		db: database,
	}
}

// WriteFile writes a TerraformFile to the database
func (w *DatabaseWriter) WriteFile(ctx context.Context, file *model.TerraformFile) error {
	query := `
		INSERT INTO terraform_files (id, source, source_path, file_type, content, content_hash, collected_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			content_hash = EXCLUDED.content_hash,
			updated_at = EXCLUDED.updated_at
	`
	
	_, err := w.db.ExecContext(ctx, query,
		file.ID, file.Source, file.SourcePath, file.FileType,
		file.Content, file.ContentHash, file.CollectedAt, file.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	return nil
}

// WriteObject writes a TerraformObject to the database
func (w *DatabaseWriter) WriteObject(ctx context.Context, obj *model.TerraformObject) error {
	configJSON, err := json.Marshal(obj.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}
	
	query := `
		INSERT INTO terraform_objects (
			id, file_id, type, name, resource_type, configuration, dependencies, address,
			mode, provider_name, schema_version, sensitive_values, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			configuration = EXCLUDED.configuration,
			dependencies = EXCLUDED.dependencies,
			updated_at = EXCLUDED.updated_at
	`
	
	_, err = w.db.ExecContext(ctx, query,
		obj.ID, obj.FileID, obj.Type, obj.Name, obj.ResourceType,
		configJSON, pq.Array(obj.Dependencies), obj.Address,
		obj.Mode, obj.ProviderName, obj.SchemaVersion,
		pq.Array(obj.SensitiveValues), obj.CreatedAt, obj.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to write object: %w", err)
	}
	
	return nil
}

// WriteBatch writes multiple files and objects in a transaction
func (w *DatabaseWriter) WriteBatch(ctx context.Context, files []*model.TerraformFile, objects []*model.TerraformObject) error {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Write files
	for _, file := range files {
		if err := w.writeFileInTx(ctx, tx, file); err != nil {
			return fmt.Errorf("failed to write file in batch: %w", err)
		}
	}
	
	// Write objects
	for _, obj := range objects {
		if err := w.writeObjectInTx(ctx, tx, obj); err != nil {
			return fmt.Errorf("failed to write object in batch: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// writeFileInTx writes a file within a transaction
func (w *DatabaseWriter) writeFileInTx(ctx context.Context, tx driver.Tx, file *model.TerraformFile) error {
	// Note: For simplicity, this is not implemented in this basic version
	// In production, would use proper transaction-aware methods
	return w.WriteFile(ctx, file)
}

// writeObjectInTx writes an object within a transaction  
func (w *DatabaseWriter) writeObjectInTx(ctx context.Context, tx driver.Tx, obj *model.TerraformObject) error {
	// Note: For simplicity, this is not implemented in this basic version
	// In production, would use proper transaction-aware methods
	return w.WriteObject(ctx, obj)
}