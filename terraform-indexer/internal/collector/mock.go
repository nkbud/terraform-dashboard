package collector

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/model"
)

// MockCollector is a simple collector for testing that returns mock data
type MockCollector struct {
	*BaseCollector
}

// NewMockCollector creates a new mock collector
func NewMockCollector(name string, source model.FileSource) *MockCollector {
	return &MockCollector{
		BaseCollector: NewBaseCollector(name, source),
	}
}

// Collect returns mock Terraform files for testing
func (c *MockCollector) Collect(ctx context.Context) ([]*model.TerraformFile, error) {
	var files []*model.TerraformFile
	
	// Try to read from testdata directory for more realistic testing
	tfContent := `resource "aws_instance" "example" {
  ami           = "ami-0c55b159cbfafe1d0"
  instance_type = "t2.micro"
  
  tags = {
    Name = "HelloWorld"
  }
}`
	
	stateContent := `{
  "version": 4,
  "terraform_version": "1.0.0",
  "serial": 1,
  "lineage": "test-lineage",
  "outputs": {},
  "resources": [
    {
      "mode": "managed",
      "type": "aws_instance", 
      "name": "example",
      "provider": "provider[\"registry.terraform.io/hashicorp/aws\"]",
      "instances": [
        {
          "schema_version": 1,
          "attributes": {
            "ami": "ami-0c55b159cbfafe1d0",
            "instance_type": "t2.micro",
            "tags": {
              "Name": "HelloWorld"
            }
          },
          "sensitive_attributes": []
        }
      ]
    }
  ]
}`

	// Create mock files based on source type
	switch c.source {
	case model.FileSourceS3:
		// Mock S3 files
		tfFile := createTerraformFile(c.source, "s3://bucket/main.tf", tfContent, model.FileTypeTerraform)
		files = append(files, tfFile)
		
		stateFile := createTerraformFile(c.source, "s3://bucket/terraform.tfstate", stateContent, model.FileTypeState)
		files = append(files, stateFile)
		
	case model.FileSourceKubernetes:
		// Mock Kubernetes files
		tfFile := createTerraformFile(c.source, "k8s://namespace/configmap/terraform-config", tfContent, model.FileTypeTerraform)
		files = append(files, tfFile)
		
	case model.FileSourceBitbucket:
		// Mock Bitbucket files
		tfFile := createTerraformFile(c.source, "bitbucket://repo/main.tf", tfContent, model.FileTypeTerraform)
		files = append(files, tfFile)
		
	default:
		// Default mock files
		tfFile := createTerraformFile(c.source, "main.tf", tfContent, model.FileTypeTerraform)
		files = append(files, tfFile)
		
		stateFile := createTerraformFile(c.source, "terraform.tfstate", stateContent, model.FileTypeState)
		files = append(files, stateFile)
	}
	
	return files, nil
}

// determineFileType determines the file type based on the file extension
func determineFileType(path string) model.FileType {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".tf":
		return model.FileTypeTerraform
	case ".tfstate":
		return model.FileTypeState
	default:
		// Check if it's a state file without extension or with .json
		if strings.Contains(path, "tfstate") || strings.HasSuffix(path, ".json") {
			return model.FileTypeState
		}
		return model.FileTypeTerraform
	}
}