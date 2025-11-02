package decorator

import (
	"fmt"
	"regexp"

	"github.com/aledsdavies/opal/core/types"
)

// ParamBuilder provides a fluent API for building parameters with type-specific constraints.
// It returns to the parent DescriptorBuilder when Done() is called.
type ParamBuilder struct {
	parent           *DescriptorBuilder
	schema           types.ParamSchema
	requiredExplicit bool // Track if Required() was explicitly called
}

// Required marks the parameter as required.
func (pb *ParamBuilder) Required() *ParamBuilder {
	pb.schema.Required = true
	pb.requiredExplicit = true
	return pb
}

// Default sets the default value for the parameter.
// Automatically marks the parameter as optional.
func (pb *ParamBuilder) Default(value any) *ParamBuilder {
	pb.schema.Default = value
	pb.schema.Required = false
	return pb
}

// Examples adds example values for documentation.
func (pb *ParamBuilder) Examples(examples ...string) *ParamBuilder {
	pb.schema.Examples = examples
	return pb
}

// MinLength sets the minimum length constraint (for strings and arrays).
func (pb *ParamBuilder) MinLength(n int) *ParamBuilder {
	pb.schema.MinLength = &n
	return pb
}

// MaxLength sets the maximum length constraint (for strings and arrays).
func (pb *ParamBuilder) MaxLength(n int) *ParamBuilder {
	pb.schema.MaxLength = &n
	return pb
}

// Pattern sets a regex pattern constraint (for strings).
// Pattern is validated when Build() is called.
func (pb *ParamBuilder) Pattern(regex string) *ParamBuilder {
	pb.schema.Pattern = &regex
	return pb
}

// Format sets a typed format constraint (for strings).
// Examples: FormatURI, FormatHostname, FormatIPv4, FormatCIDR, FormatSemver, FormatDuration
func (pb *ParamBuilder) Format(format types.Format) *ParamBuilder {
	pb.schema.Format = &format
	return pb
}

// Min sets the minimum value constraint (for numeric types).
func (pb *ParamBuilder) Min(min float64) *ParamBuilder {
	pb.schema.Minimum = &min
	return pb
}

// Max sets the maximum value constraint (for numeric types).
func (pb *ParamBuilder) Max(max float64) *ParamBuilder {
	pb.schema.Maximum = &max
	return pb
}

// Done finishes building this parameter and returns to the parent DescriptorBuilder.
// Validates the parameter schema before adding it.
func (pb *ParamBuilder) Done() *DescriptorBuilder {
	// Validate parameter schema
	if err := pb.validate(); err != nil {
		panic(fmt.Sprintf("invalid parameter %q: %v", pb.schema.Name, err))
	}

	// Add parameter to parent descriptor
	pb.parent.desc.Schema.Parameters[pb.schema.Name] = pb.schema
	pb.parent.desc.Schema.ParameterOrder = append(pb.parent.desc.Schema.ParameterOrder, pb.schema.Name)

	return pb.parent
}

// validate checks parameter schema for common errors.
func (pb *ParamBuilder) validate() error {
	// Check for required + default (invalid combination)
	// Only error if Required() was explicitly called before Default()
	if pb.requiredExplicit && pb.schema.Default != nil {
		return fmt.Errorf("parameter cannot be both required and have a default value")
	}

	// Validate regex pattern if set
	if pb.schema.Pattern != nil {
		if _, err := regexp.Compile(*pb.schema.Pattern); err != nil {
			return fmt.Errorf("invalid regex pattern %q: %w", *pb.schema.Pattern, err)
		}
	}

	// Validate min <= max for numeric types
	if pb.schema.Minimum != nil && pb.schema.Maximum != nil {
		if *pb.schema.Minimum > *pb.schema.Maximum {
			return fmt.Errorf("minimum (%v) cannot be greater than maximum (%v)", *pb.schema.Minimum, *pb.schema.Maximum)
		}
	}

	return nil
}

// ParamString creates a string parameter builder.
func (b *DescriptorBuilder) ParamString(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeString,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// ParamInt creates an integer parameter builder.
func (b *DescriptorBuilder) ParamInt(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeInt,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// ParamFloat creates a float parameter builder.
func (b *DescriptorBuilder) ParamFloat(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeFloat,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// ParamBool creates a boolean parameter builder.
func (b *DescriptorBuilder) ParamBool(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeBool,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// ParamDuration creates a duration parameter builder.
func (b *DescriptorBuilder) ParamDuration(name, description string) *ParamBuilder {
	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeDuration,
			Description: description,
			Required:    false, // Optional by default
		},
	}
}

// PrimaryParamString creates a string primary parameter builder.
// Primary parameters are always required and appear first in parameter order.
func (b *DescriptorBuilder) PrimaryParamString(name, description string) *ParamBuilder {
	// Check for duplicate primary parameter
	if b.desc.Schema.PrimaryParameter != "" {
		panic(fmt.Sprintf("primary parameter already set to %q, cannot set to %q", b.desc.Schema.PrimaryParameter, name))
	}

	b.desc.Schema.PrimaryParameter = name

	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeString,
			Description: description,
			Required:    true, // Primary parameters are always required
		},
	}
}

// PrimaryParamInt creates an integer primary parameter builder.
func (b *DescriptorBuilder) PrimaryParamInt(name, description string) *ParamBuilder {
	// Check for duplicate primary parameter
	if b.desc.Schema.PrimaryParameter != "" {
		panic(fmt.Sprintf("primary parameter already set to %q, cannot set to %q", b.desc.Schema.PrimaryParameter, name))
	}

	b.desc.Schema.PrimaryParameter = name

	return &ParamBuilder{
		parent: b,
		schema: types.ParamSchema{
			Name:        name,
			Type:        types.TypeInt,
			Description: description,
			Required:    true, // Primary parameters are always required
		},
	}
}
