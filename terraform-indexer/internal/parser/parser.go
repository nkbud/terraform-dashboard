package parser

import (
	"context"

	"github.com/nkbud/terraform-dashboard/terraform-indexer/internal/model"
)

// Parser defines the interface for parsing Terraform files
type Parser interface {
	// Parse parses a TerraformFile and returns TerraformObjects
	Parse(ctx context.Context, file *model.TerraformFile) ([]*model.TerraformObject, error)
	
	// CanParse returns true if this parser can handle the given file type
	CanParse(fileType model.FileType) bool
}

// ParserRegistry manages multiple parsers
type ParserRegistry struct {
	parsers []Parser
}

// NewParserRegistry creates a new parser registry
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make([]Parser, 0),
	}
}

// Register adds a parser to the registry
func (r *ParserRegistry) Register(parser Parser) {
	r.parsers = append(r.parsers, parser)
}

// Parse finds the appropriate parser and parses the file
func (r *ParserRegistry) Parse(ctx context.Context, file *model.TerraformFile) ([]*model.TerraformObject, error) {
	for _, parser := range r.parsers {
		if parser.CanParse(file.FileType) {
			return parser.Parse(ctx, file)
		}
	}
	
	// If no specific parser found, return empty slice
	return []*model.TerraformObject{}, nil
}