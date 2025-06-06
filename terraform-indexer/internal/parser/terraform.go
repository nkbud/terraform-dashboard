package parser

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/model"
)

// TerraformParser parses Terraform configuration files
type TerraformParser struct{}

// NewTerraformParser creates a new Terraform parser
func NewTerraformParser() *TerraformParser {
	return &TerraformParser{}
}

// CanParse returns true if this parser can handle the file type
func (p *TerraformParser) CanParse(fileType model.FileType) bool {
	return fileType == model.FileTypeTerraform
}

// Parse parses a Terraform configuration file
func (p *TerraformParser) Parse(ctx context.Context, file *model.TerraformFile) ([]*model.TerraformObject, error) {
	parser := hclparse.NewParser()
	
	hclFile, diags := parser.ParseHCL([]byte(file.Content), file.SourcePath)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}
	
	var objects []*model.TerraformObject
	
	// Parse the body content
	content, _, diags := hclFile.Body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
			{Type: "provider", LabelNames: []string{"name"}},
			{Type: "variable", LabelNames: []string{"name"}},
			{Type: "output", LabelNames: []string{"name"}},
			{Type: "module", LabelNames: []string{"name"}},
			{Type: "data", LabelNames: []string{"type", "name"}},
		},
	})
	
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse content: %s", diags.Error())
	}
	
	// Process each block
	for _, block := range content.Blocks {
		obj, err := p.parseBlock(file.ID, block)
		if err != nil {
			// Log error but continue parsing other blocks
			continue
		}
		objects = append(objects, obj)
	}
	
	return objects, nil
}

// parseBlock parses an HCL block into a TerraformObject
func (p *TerraformParser) parseBlock(fileID string, block *hcl.Block) (*model.TerraformObject, error) {
	var objectType model.TerraformObjectType
	var name, resourceType, address string
	
	switch block.Type {
	case "resource":
		objectType = model.ObjectTypeResource
		resourceType = block.Labels[0]
		name = block.Labels[1]
		address = fmt.Sprintf("%s.%s", resourceType, name)
	case "provider":
		objectType = model.ObjectTypeProvider
		name = block.Labels[0]
		address = fmt.Sprintf("provider.%s", name)
	case "variable":
		objectType = model.ObjectTypeVariable
		name = block.Labels[0]
		address = fmt.Sprintf("var.%s", name)
	case "output":
		objectType = model.ObjectTypeOutput
		name = block.Labels[0]
		address = fmt.Sprintf("output.%s", name)
	case "module":
		objectType = model.ObjectTypeModule
		name = block.Labels[0]
		address = fmt.Sprintf("module.%s", name)
	case "data":
		objectType = model.ObjectTypeData
		resourceType = block.Labels[0]
		name = block.Labels[1]
		address = fmt.Sprintf("data.%s.%s", resourceType, name)
	default:
		return nil, fmt.Errorf("unknown block type: %s", block.Type)
	}
	
	// Extract configuration from block body
	config, err := p.extractConfiguration(block.Body)
	if err != nil {
		config = make(map[string]interface{}) // Use empty config on error
	}
	
	obj := &model.TerraformObject{
		ID:            generateObjectID(fileID, string(objectType), name, 0),
		FileID:        fileID,
		Type:          objectType,
		Name:          name,
		ResourceType:  resourceType,
		Configuration: config,
		Address:       address,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	return obj, nil
}

// extractConfiguration extracts configuration from an HCL body
func (p *TerraformParser) extractConfiguration(body hcl.Body) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	
	// Get all attributes
	attrs, diags := body.JustAttributes()
	if diags.HasErrors() {
		return config, fmt.Errorf("failed to get attributes: %s", diags.Error())
	}
	
	for name, attr := range attrs {
		// Simple value extraction - in production this would be more sophisticated
		val, err := p.extractSimpleValue(attr.Expr)
		if err == nil {
			config[name] = val
		} else {
			// Store as string representation if we can't extract the value
			config[name] = fmt.Sprintf("%v", attr.Expr)
		}
	}
	
	return config, nil
}

// extractSimpleValue attempts to extract simple values from HCL expressions
func (p *TerraformParser) extractSimpleValue(expr hcl.Expression) (interface{}, error) {
	// This is a simplified implementation - a full implementation would handle
	// all HCL expression types, variable references, functions, etc.
	
	// For now, return the expression as a string representation
	return fmt.Sprintf("%v", expr), nil
}