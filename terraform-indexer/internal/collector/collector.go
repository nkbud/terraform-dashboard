package collector

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/model"
)

// Collector defines the interface for collecting Terraform files from various sources
type Collector interface {
	// Name returns the name of the collector
	Name() string
	
	// Collect polls for files and returns collected files
	Collect(ctx context.Context) ([]*model.TerraformFile, error)
	
	// Source returns the source type this collector handles
	Source() model.FileSource
}

// CollectorConfig represents configuration for collectors
type CollectorConfig struct {
	Name     string                 `yaml:"name"`
	Type     string                 `yaml:"type"`
	Settings map[string]interface{} `yaml:"settings"`
	Enabled  bool                   `yaml:"enabled"`
}

// BaseCollector provides common functionality for all collectors
type BaseCollector struct {
	name   string
	source model.FileSource
}

// NewBaseCollector creates a new base collector
func NewBaseCollector(name string, source model.FileSource) *BaseCollector {
	return &BaseCollector{
		name:   name,
		source: source,
	}
}

// Name returns the collector name
func (c *BaseCollector) Name() string {
	return c.name
}

// Source returns the source type
func (c *BaseCollector) Source() model.FileSource {
	return c.source
}

// generateFileID generates a unique ID for a file based on source and path
func generateFileID(source model.FileSource, path string) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s", source, path)))
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}

// generateContentHash generates a hash of the file content
func generateContentHash(content string) string {
	h := sha256.New()
	h.Write([]byte(content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// createTerraformFile creates a TerraformFile from the given parameters
func createTerraformFile(source model.FileSource, path, content string, fileType model.FileType) *model.TerraformFile {
	now := time.Now()
	return &model.TerraformFile{
		ID:          generateFileID(source, path),
		Source:      source,
		SourcePath:  path,
		FileType:    fileType,
		Content:     content,
		ContentHash: generateContentHash(content),
		CollectedAt: now,
		UpdatedAt:   now,
	}
}