package convert

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ObjectConverter provides utilities for nested object conversion
type ObjectConverter struct {
	ctx context.Context
}

// NewObjectConverter creates a new ObjectConverter with the given context
func NewObjectConverter(ctx context.Context) *ObjectConverter {
	return &ObjectConverter{ctx: ctx}
}

// ToObject converts a struct to types.Object
func (c *ObjectConverter) ToObject(attrTypes map[string]attr.Type, value interface{}) (types.Object, diag.Diagnostics) {
	return types.ObjectValueFrom(c.ctx, attrTypes, value)
}

// ToList converts a slice to types.List
func (c *ObjectConverter) ToList(elementType attr.Type, values interface{}) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(c.ctx, elementType, values)
}
