package parser

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/model"
)

// StateParser parses Terraform state files
type StateParser struct{}

// NewStateParser creates a new state parser
func NewStateParser() *StateParser {
	return &StateParser{}
}

// CanParse returns true if this parser can handle the file type
func (p *StateParser) CanParse(fileType model.FileType) bool {
	return fileType == model.FileTypeState
}

// Parse parses a Terraform state file
func (p *StateParser) Parse(ctx context.Context, file *model.TerraformFile) ([]*model.TerraformObject, error) {
	var state TerraformState
	if err := json.Unmarshal([]byte(file.Content), &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	
	var objects []*model.TerraformObject
	
	for _, resource := range state.Resources {
		for i, instance := range resource.Instances {
			obj := &model.TerraformObject{
				ID:            generateObjectID(file.ID, resource.Type, resource.Name, i),
				FileID:        file.ID,
				Type:          model.ObjectTypeResource,
				Name:          resource.Name,
				ResourceType:  resource.Type,
				Configuration: instance.Attributes,
				Address:       fmt.Sprintf("%s.%s", resource.Type, resource.Name),
				Mode:          resource.Mode,
				ProviderName:  extractProviderName(resource.Provider),
				SchemaVersion: instance.SchemaVersion,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			
			objects = append(objects, obj)
		}
	}
	
	return objects, nil
}

// TerraformState represents the structure of a Terraform state file
type TerraformState struct {
	Version          int                    `json:"version"`
	TerraformVersion string                 `json:"terraform_version"`
	Serial           int                    `json:"serial"`
	Lineage          string                 `json:"lineage"`
	Outputs          map[string]interface{} `json:"outputs"`
	Resources        []StateResource        `json:"resources"`
}

// StateResource represents a resource in the Terraform state
type StateResource struct {
	Mode      string          `json:"mode"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Provider  string          `json:"provider"`
	Instances []StateInstance `json:"instances"`
}

// StateInstance represents an instance of a resource in the state
type StateInstance struct {
	SchemaVersion    int                    `json:"schema_version"`
	Attributes       map[string]interface{} `json:"attributes"`
	SensitiveAttrs   []string               `json:"sensitive_attributes,omitempty"`
	Dependencies     []string               `json:"dependencies,omitempty"`
}

// generateObjectID generates a unique ID for a TerraformObject
func generateObjectID(fileID, resourceType, name string, index int) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s:%s:%d", fileID, resourceType, name, index)))
	return fmt.Sprintf("%x", h.Sum(nil)[:8])
}

// extractProviderName extracts the provider name from the provider string
func extractProviderName(provider string) string {
	// Extract from provider["registry.terraform.io/hashicorp/aws"] -> aws
	if len(provider) == 0 {
		return ""
	}
	
	// Simple extraction - in real implementation this would be more robust
	start := len(provider) - 1
	for i := len(provider) - 1; i >= 0; i-- {
		if provider[i] == '/' {
			start = i + 1
			break
		}
	}
	
	end := len(provider)
	for i := start; i < len(provider); i++ {
		if provider[i] == '"' || provider[i] == ']' {
			end = i
			break
		}
	}
	
	if start < end {
		return provider[start:end]
	}
	
	return provider
}