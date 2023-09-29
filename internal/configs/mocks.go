// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
)

// MockData contains sets of values for resources and data sources that are
// mocked by a given provider.
type MockData struct {
	Resources   map[string]*MockResource
	DataSources map[string]*MockResource
}

// MockResource maps a resource or data source type and name to a set of values
// for that resource.
type MockResource struct {
	Mode addrs.ResourceMode
	Type string

	TypeRange hcl.Range

	Defaults  cty.Value
	Overrides addrs.Map[addrs.Targetable, *ResourceOverride]
}

// ResourceOverride targets a specific module, resource, or data source with
// replacement values that should be used in place of whatever the provider
// would normally do.
type ResourceOverride struct {
	TypeRange hcl.Range

	Address *addrs.Target
	Values  cty.Value

	DeclRange hcl.Range
}

func decodeMockDataBody(body hcl.Body) (*MockData, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := body.Content(mockDataSchema)
	diags = append(diags, contentDiags...)

	d := MockData{
		Resources:   make(map[string]*MockResource),
		DataSources: make(map[string]*MockResource),
	}

	for _, block := range content.Blocks {

		resource, resourceDiags := decodeMockResourceBlock(block)
		diags = append(diags, resourceDiags...)

		switch resource.Mode {
		case addrs.ManagedResourceMode:
			if previous, ok := d.Resources[resource.Type]; ok {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate resource block",
					Detail:   fmt.Sprintf("A resource block for %s has already been defined at %s.", resource.Type, previous.TypeRange.String()),
					Subject:  resource.TypeRange.Ptr(),
				})
				continue
			}
			d.Resources[resource.Type] = resource
		case addrs.DataResourceMode:
			if previous, ok := d.DataSources[resource.Type]; ok {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate data block",
					Detail:   fmt.Sprintf("A data block for %s has already been defined at %s.", resource.Type, previous.TypeRange.String()),
					Subject:  resource.TypeRange.Ptr(),
				})
				continue
			}
			d.DataSources[resource.Type] = resource
		}

	}

	return &d, diags
}

func decodeMockResourceBlock(block *hcl.Block) (*MockResource, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := block.Body.Content(mockResourceSchema)
	diags = append(diags, contentDiags...)

	r := MockResource{
		Type:      block.Labels[0],
		TypeRange: block.TypeRange,

		Overrides: addrs.MakeMap[addrs.Targetable, *ResourceOverride](),

		Defaults: cty.NilVal, // This is optional, so we'll set the default.
	}

	switch block.Type {
	case "resource":
		r.Mode = addrs.ManagedResourceMode
	case "data":
		r.Mode = addrs.DataResourceMode
	}

	for _, overrideBlock := range content.Blocks {
		// The only block our schema supports is the override block.
		override, overrideDiags := decodeOverrideBlock(overrideBlock, false)
		diags = append(diags, overrideDiags...)

		if !overrideDiags.HasErrors() {
			if existing, ok := r.Overrides.GetOk(override.Address.Subject); ok {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate override block",
					Detail:   fmt.Sprintf("An override block targeting %s has already been defined at %s.", existing.Address.Subject, existing.TypeRange),
					Subject:  override.TypeRange.Ptr(),
				})
				continue
			}

			var addr addrs.AbsResource
			switch override.Address.Subject.AddrType() {
			case addrs.AbsResourceAddrType:
				addr = override.Address.Subject.(addrs.AbsResource)
			case addrs.AbsResourceInstanceAddrType:
				addr = override.Address.Subject.(addrs.AbsResourceInstance).ContainingResource()
			}

			resource := addr.Resource.Type

			var mode string
			switch addr.Resource.Mode {
			case addrs.DataResourceMode:
				mode = "data"
			case addrs.ManagedResourceMode:
				mode = "resource"
			}

			if resource != r.Type {
				// Then this address is targeting the wrong type.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid resource type",
					Detail:   fmt.Sprintf("You have targeted resource type %q for an override while defining resource type %q.", resource, r.Type),
					Subject:  override.Address.SourceRange.ToHCL().Ptr(),
				})
			}

			if mode != block.Type {
				// Then this address is targeting the wrong mode.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid resource type",
					Detail:   fmt.Sprintf("You have targeted resource mode %q for an override while defining resource type %q.", mode, block.Type),
					Subject:  override.Address.SourceRange.ToHCL().Ptr(),
				})
			}

			r.Overrides.Put(override.Address.Subject, override)
		}
	}

	if attr, exists := content.Attributes["defaults"]; exists {
		defaults, defaultDiags := attr.Expr.Value(nil)
		diags = append(diags, defaultDiags...)
		r.Defaults = defaults
	}

	return &r, diags
}

func decodeOverrideBlock(block *hcl.Block, allowModules bool) (*ResourceOverride, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := block.Body.Content(resourceOverrideSchema)
	diags = append(diags, contentDiags...)

	o := ResourceOverride{
		TypeRange: block.TypeRange,
	}

	if attr, exists := content.Attributes["addr"]; exists {
		traversal, traversalDiags := hcl.AbsTraversalForExpr(attr.Expr)
		diags = append(diags, traversalDiags...)
		if !traversalDiags.HasErrors() {
			target, targetDiags := addrs.ParseTarget(traversal)
			diags = append(diags, targetDiags.ToHCL()...)
			o.Address = target
		}
	} else {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing \"addr\" attribute",
			Detail:   "Override blocks must specify a target address.",
			Subject:  block.TypeRange.Ptr(),
		})
	}

	if attr, exists := content.Attributes["values"]; exists {
		values, valueDiags := attr.Expr.Value(nil)
		diags = append(diags, valueDiags...)
		o.Values = values
	} else {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing \"values\" attribute",
			Detail:   "Override blocks must specify the replacement values.",
			Subject:  block.TypeRange.Ptr(),
		})
	}

	if o.Address != nil {
		switch o.Address.Subject.AddrType() {
		case addrs.ConfigResourceAddrType:
			// We can't process this kind of address later, but equally the
			// ParseTarget function never returns it! Add this is in for
			// sanity's sake and it makes later validation easier.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid override address target",
				Detail:   fmt.Sprintf("Terraform has evaluated an override address %s as a configuration resource; This is a bug in Terraform - please report it.", o.Address.Subject),
				Subject:  o.Address.SourceRange.ToHCL().Ptr(),
			})
		case addrs.ModuleAddrType:
			// Same as above, we shouldn't get this type back from ParseTarget.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid override address target",
				Detail:   fmt.Sprintf("Terraform has evaluated an override address %s as a configuration module; This is a bug in Terraform - please report it.", o.Address.Subject),
				Subject:  o.Address.SourceRange.ToHCL().Ptr(),
			})
		case addrs.ModuleInstanceAddrType:
			// You can only specify modules if allowModules is true.
			if !allowModules {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid override address target",
					Detail:   fmt.Sprintf("A module target %s is not acceptable in this context. Modules can only be targeted by override blocks defined directly within test files or test file run blocks.", o.Address.Subject),
					Subject:  o.Address.SourceRange.ToHCL().Ptr(),
				})
			}
		case addrs.AbsResourceInstanceAddrType:
			// No diagnostics here, these are always okay!
		case addrs.AbsResourceAddrType:
			addr := o.Address.Subject.(addrs.AbsResource)

			// Turn this into an instance. Later, when we look things up we
			// always look up by instanced addresses.
			o.Address = &addrs.Target{
				Subject: addrs.AbsResourceInstance{
					Module:   addr.Module,
					Resource: addr.Resource.Instance(addrs.NoKey),
				},
				SourceRange: o.Address.SourceRange,
			}
		default:
			// Future-proof this, we need to explicitly mark types as being
			// acceptable or we'll return a generic error.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid override address target",
				Detail:   fmt.Sprintf("Terraform has found an unrecognized address type %d; This configuration may have been written targeting a later version of Terraform.", o.Address.Subject.AddrType()),
				Subject:  o.Address.SourceRange.ToHCL().Ptr(),
			})
		}
	}

	return &o, diags
}

var mockDataSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "resource",
			LabelNames: []string{"type"},
		},
		{
			Type:       "data",
			LabelNames: []string{"type"},
		},
	},
}

var mockResourceSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "defaults",
		},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "override",
		},
	},
}

var resourceOverrideSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{
			Name: "addr",
		},
		{
			Name: "values",
		},
	},
}
