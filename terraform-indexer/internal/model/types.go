package model

import (
	"time"
)

// FileSource represents the source of a Terraform file
type FileSource string

const (
	FileSourceS3         FileSource = "s3"
	FileSourceKubernetes FileSource = "kubernetes"
	FileSourceBitbucket  FileSource = "bitbucket"
)

// FileType represents the type of Terraform file
type FileType string

const (
	FileTypeTerraform FileType = "tf"
	FileTypeState     FileType = "tfstate"
)

// TerraformObjectType represents the type of Terraform object
type TerraformObjectType string

const (
	ObjectTypeResource TerraformObjectType = "resource"
	ObjectTypeProvider TerraformObjectType = "provider"
	ObjectTypeVariable TerraformObjectType = "variable"
	ObjectTypeOutput   TerraformObjectType = "output"
	ObjectTypeModule   TerraformObjectType = "module"
	ObjectTypeData     TerraformObjectType = "data"
)

// TerraformFile represents a Terraform file that has been collected
type TerraformFile struct {
	ID          string     `json:"id"`
	Source      FileSource `json:"source"`
	SourcePath  string     `json:"source_path"`
	FileType    FileType   `json:"file_type"`
	Content     string     `json:"content"`
	ContentHash string     `json:"content_hash"`
	CollectedAt time.Time  `json:"collected_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TerraformObject represents a parsed Terraform construct
type TerraformObject struct {
	ID               string              `json:"id"`
	FileID           string              `json:"file_id"`
	Type             TerraformObjectType `json:"type"`
	Name             string              `json:"name"`
	ResourceType     string              `json:"resource_type,omitempty"`
	Configuration    map[string]any      `json:"configuration"`
	Dependencies     []string            `json:"dependencies"`
	Address          string              `json:"address"`
	Mode             string              `json:"mode,omitempty"`
	ProviderName     string              `json:"provider_name,omitempty"`
	SchemaVersion    int                 `json:"schema_version,omitempty"`
	SensitiveValues  []string            `json:"sensitive_values,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}