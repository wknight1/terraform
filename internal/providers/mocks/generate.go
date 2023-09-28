// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocks

import (
	"fmt"
	"math/big"
	"math/rand"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var testRand *rand.Rand

var (
	chars = []rune("abcdefghijklmnopqrstuvwxyz0123456789")
)

type Filler interface {
	Process(original cty.Value) bool
	Fill(original, target cty.Value, path cty.Path) (cty.Value, tfdiags.Diagnostics)
}

func FillComputedValues(original, target cty.Value, schema *configschema.Block, filler Filler) (cty.Value, tfdiags.Diagnostics) {
	return fillComputedValues(original, target, schema, filler, cty.Path{})
}

func fillComputedValues(original, target cty.Value, schema *configschema.Block, filler Filler, path cty.Path) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var fmtdPath string
	if len(path) > 0 {
		fmtdPath = fmt.Sprintf("%s: ", tfdiags.FormatCtyPath(path))
	}

	if schema == nil {
		// The caller must have provided a schema for us to iterate through.
		panic("must have provided a schema; this is a bug in Terraform - please report it")
	}

	if !original.Type().IsObjectType() {
		// This means the Terraform internals have messed up somewhere as there
		// should have been validation that meant this was caught earlier.
		panic("must have provided an object type; this is a bug in Terraform - please report it")
	}

	if target != cty.NilVal && !target.Type().IsObjectType() {
		// A NilVal is okay, as it means we just don't have any defaults for
		// this value. But if we do have defaults, it must be an object type
		// and this value is provided by the user so we'll surface a real error
		// for this instead of panicking.
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Type mismatch", fmt.Sprintf("%sExpected object but found %s", fmtdPath, target.Type().FriendlyName())))
		return original, diags
	}

	// We're going to build a replacement value now. It will contain the attrs
	// from original, and anything the schema said that should be computed but
	// is null.
	attrs := make(map[string]cty.Value)
	for name, block := range schema.BlockTypes {

		// Get the value for this block from original.
		original := original.GetAttr(name)
		if original.IsNull() {
			// Then we have no block to recurse into, and since blocks can't
			// be unknown themselves we'll just use the null block.
			attrs[name] = original
			continue
		}

		// Get the value for this block from the target. This may return
		// cty.NilVal if no attrs were defined for this block in target, but
		// that's okay. We are prepared to handle target being a NilVal.
		target := getChildSafe(target, name)
		if target != cty.NilVal && !target.Type().IsObjectType() {
			// When filling in default attrs for blocks, we only allow users
			// to specify a single object type that is then used to populate
			// all the nested blocks for a given block type.
			//
			// We could try and allow users to specify specific attrs for the
			// nested blocks, but there isn't really a good way to do that for
			// sets. We have no way of matching up the users intended nested
			// block for each value they give us. I think it's therefore simpler
			// for us to just say all nested blocks are the same.
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Type mismatch", fmt.Sprintf("%sExpected attribute %s to be an object but found %s", fmtdPath, name, target.Type().FriendlyName())))
			return original, diags
		}

		switch block.Nesting {
		case configschema.NestingSingle, configschema.NestingGroup:
			if !original.Type().IsObjectType() {
				// The object provided by Terraform should match the expected
				// schema, if not then validation has failed somewhere else.
				panic(fmt.Errorf("must have provided a object type; this is a bug in Terraform - please report it"))
			}

			value, valueDiags := fillComputedValues(original, target, &block.Block, filler, path.GetAttr(name))
			diags = diags.Append(valueDiags)
			attrs[name] = value
		case configschema.NestingList:
			if !original.Type().IsListType() {
				// The object provided by Terraform should match the expected
				// schema, if not then validation has failed somewhere else.
				panic(fmt.Errorf("must have provided a list type; this is a bug in Terraform - please report it"))
			}

			if original.LengthInt() == 0 {
				// Easy case, it's empty so we have nothing to populate.
				attrs[name] = original
				continue
			}

			var values []cty.Value
			for ix, child := range original.AsValueSlice() {
				value, valueDiags := fillComputedValues(child, target, &block.Block, filler, path.GetAttr(name).IndexInt(ix))
				diags = diags.Append(valueDiags)
				values = append(values, value)
			}
			attrs[name] = cty.ListVal(values)

		case configschema.NestingMap:
			if !original.Type().IsMapType() {
				// The object provided by Terraform should match the expected
				// schema, if not then validation has failed somewhere else.
				panic(fmt.Errorf("must have provided a map type; this is a bug in Terraform - please report it"))
			}

			if original.LengthInt() == 0 {
				// Easy case, it's empty so we have nothing to populate.
				attrs[name] = original
				continue
			}

			values := make(map[string]cty.Value)
			for key, child := range original.AsValueMap() {
				value, valueDiags := fillComputedValues(child, target, &block.Block, filler, path.GetAttr(name).IndexString(key))
				diags = diags.Append(valueDiags)
				values[key] = value
			}
			attrs[name] = cty.MapVal(values)

		case configschema.NestingSet:
			if !original.Type().IsSetType() {
				// The object provided by Terraform should match the expected
				// schema, if not then validation has failed somewhere else.
				panic(fmt.Errorf("must have provided a set type; this is a bug in Terraform - please report it"))
			}

			if original.LengthInt() == 0 {
				// Easy case, it's empty so we have nothing to populate.
				attrs[name] = original
				continue
			}

			var values []cty.Value
			for _, child := range original.AsValueSlice() {
				value, valueDiags := fillComputedValues(child, target, &block.Block, filler, path.GetAttr(name).IndexString(child.GoString()))
				diags = diags.Append(valueDiags)
				values = append(values, value)
			}
			attrs[name] = cty.SetVal(values)

		default:
			panic(fmt.Errorf("unknown nesting mode: %s", block.Nesting))
		}

	}

	for name, attribute := range schema.Attributes {
		child, childDiags := fillComputedValuesForAttribute(original.GetAttr(name), getChildSafe(target, name), attribute, filler, path.GetAttr(name))
		diags = diags.Append(childDiags)
		attrs[name] = child
	}

	if len(attrs) == 0 {
		return cty.EmptyObjectVal, diags
	}
	return cty.ObjectVal(attrs), diags
}

func fillComputedValuesForAttribute(original, target cty.Value, schema *configschema.Attribute, filler Filler, path cty.Path) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First, check if we should handle this value via the Filler.
	if schema.Computed && filler.Process(original) {
		return filler.Fill(original, target, path)
	}

	var fmtdPath string
	if len(path) > 0 {
		fmtdPath = fmt.Sprintf("%s: ", tfdiags.FormatCtyPath(path))
	}

	// If we get here, then we didn't need to do anything for this value at the
	// current level. However, nested attributes might contain nested computed
	// values which we need to look for.
	if schema.NestedType != nil {
		if original.IsNull() {
			// We know this value doesn't need to be processed by the Filler and
			// because it's null it doesn't have any child attributes that might
			// need processing so just return it as is.
			return original, diags
		}

		if target != cty.NilVal && !target.Type().IsObjectType() {
			// When filling in attributes for nested attributes, we only allow
			// users to specify a single object type that is then used to
			// populate all the nested child attributes.
			//
			// We could try and allow users to specify specific attrs for the
			// nested blocks, but there isn't really a good way to do that for
			// sets. We have no way of matching up the users intended nested
			// block for each value they give us. I think it's therefore simpler
			// for us to just say all nested blocks are the same.
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Type mismatch", fmt.Sprintf("%sExpected object but found %s", fmtdPath, target.Type().FriendlyName())))
			return original, diags
		}

		switch schema.NestedType.Nesting {
		case configschema.NestingSingle, configschema.NestingGroup:
			if !original.Type().IsObjectType() {
				// The object provided by Terraform should match the expected
				// schema, if not then validation has failed somewhere else.
				panic(fmt.Errorf("must have provided a object type; this is a bug in Terraform - please report it"))
			}

			attrs := make(map[string]cty.Value)
			for name, attr := range schema.NestedType.Attributes {
				child, childDiags := fillComputedValuesForAttribute(original.GetAttr(name), getChildSafe(target, name), attr, filler, path.GetAttr(name))
				diags = diags.Append(childDiags)
				attrs[name] = child
			}
			return cty.ObjectVal(attrs), diags

		case configschema.NestingList:
			if !original.Type().IsListType() {
				// The object provided by Terraform should match the expected
				// schema, if not then validation has failed somewhere else.
				panic(fmt.Errorf("must have provided a list type; this is a bug in Terraform - please report it"))
			}

			if original.LengthInt() == 0 {
				// Easy case, it's empty so we have nothing to populate.
				return original, nil
			}

			var values []cty.Value
			for ix, child := range original.AsValueSlice() {
				attributes := make(map[string]cty.Value)
				for name, attribute := range schema.NestedType.Attributes {
					child, childDiags := fillComputedValuesForAttribute(child.GetAttr(name), getChildSafe(target, name), attribute, filler, path.IndexInt(ix).GetAttr(name))
					diags = diags.Append(childDiags)
					attributes[name] = child
				}
				values = append(values, cty.ObjectVal(attributes))
			}
			return cty.ListVal(values), diags

		case configschema.NestingMap:
			if !original.Type().IsMapType() {
				// The object provided by Terraform should match the expected
				// schema, if not then validation has failed somewhere else.
				panic(fmt.Errorf("must have provided a map type; this is a bug in Terraform - please report it"))
			}

			if original.LengthInt() == 0 {
				// Easy case, it's empty so we have nothing to populate.
				return original, nil
			}

			values := make(map[string]cty.Value)
			for key, child := range original.AsValueMap() {
				attributes := make(map[string]cty.Value)
				for name, attribute := range schema.NestedType.Attributes {
					child, childDiags := fillComputedValuesForAttribute(child.GetAttr(name), getChildSafe(target, name), attribute, filler, path.IndexString(key).GetAttr(name))
					diags = diags.Append(childDiags)
					attributes[name] = child
				}
				values[key] = cty.ObjectVal(attributes)
			}
			return cty.MapVal(values), diags

		case configschema.NestingSet:
			if !original.Type().IsSetType() {
				// The object provided by Terraform should match the expected
				// schema, if not then validation has failed somewhere else.
				panic(fmt.Errorf("must have provided a set type; this is a bug in Terraform - please report it"))
			}

			if original.LengthInt() == 0 {
				// Easy case, it's empty so we have nothing to populate.
				return original, nil
			}

			var values []cty.Value
			for _, child := range original.AsValueSlice() {
				attributes := make(map[string]cty.Value)
				for name, attribute := range schema.NestedType.Attributes {
					child, childDiags := fillComputedValuesForAttribute(child.GetAttr(name), getChildSafe(target, name), attribute, filler, path.IndexString(child.GoString()).GetAttr(name))
					diags = diags.Append(childDiags)
					attributes[name] = child
				}
				values = append(values, cty.ObjectVal(attributes))
			}
			return cty.SetVal(values), diags

		default:
			panic(fmt.Errorf("unknown nesting mode: %s", schema.NestedType.Nesting))
		}
	}

	// Otherwise, it's not a nested type so there won't be nested computed
	// attributes.
	return original, nil
}

// UnknownFiller replaces original with cty.UnknownVal is original is null. It
// returns original otherwise. The other value doesn't matter, we just ignore
// it.
func UnknownFiller() Filler {
	return &unknownFiller{}
}

type unknownFiller struct{}

func (u *unknownFiller) Process(original cty.Value) bool {
	return original.IsNull()
}

func (u *unknownFiller) Fill(original, _ cty.Value, _ cty.Path) (cty.Value, tfdiags.Diagnostics) {
	return cty.UnknownVal(original.Type()), nil
}

// ValueFiller will attempt to populate original based on target. If original
// is already known, then this function just returns original. If original is
// unknown and target is valid (ie. the types match and it is not nil then
// target is returned. If original is unknown and target is not valid, then a
// new value will be created using junk data and returned.
func ValueFiller() Filler {
	return &valueFiller{}
}

type valueFiller struct{}

func (v *valueFiller) Process(original cty.Value) bool {
	return !original.IsKnown()
}

func (v *valueFiller) Fill(original, target cty.Value, path cty.Path) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	var fmtdPath string
	if len(path) > 0 {
		fmtdPath = fmt.Sprintf("%s: ", tfdiags.FormatCtyPath(path))
	}

	if target != cty.NilVal {
		value, err := convert.Convert(target, original.Type())
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Type mismatch", fmt.Sprintf("%sFailed to convert the provided value into the required value: %v", fmtdPath, err)))
			return original, diags
		}
		return value, diags
	}

	switch {
	case original.Type().IsPrimitiveType():
		switch original.Type() {
		case cty.String:
			// Return a random 8 character string for strings.
			return cty.StringVal(str(8)), nil
		case cty.Number:
			// Return 0 for generate numbers.
			return cty.NumberVal(big.NewFloat(0)), nil
		case cty.Bool:
			// Return false for generated booleans.
			return cty.False, nil
		default:
			panic(fmt.Errorf("unknown primitive type: %s", original.Type().FriendlyName()))
		}
	case original.Type().IsMapType():
		// Return an empty map for maps.
		return cty.MapValEmpty(original.Type().ElementType()), nil
	case original.Type().IsSetType():
		// Return an empty set for sets.
		return cty.SetValEmpty(original.Type().ElementType()), nil
	case original.Type().IsListType():
		// Return an empty list for lists.
		return cty.ListValEmpty(original.Type().ElementType()), nil
	case original.Type().IsObjectType():
		// An object value is complicated, as we need to generate values for
		// all child attributes of the object. We can't just return an empty
		// object like we can with collections.
		attrs := make(map[string]cty.Value)
		for name, attr := range original.Type().AttributeTypes() {
			child, childDiags := v.Fill(cty.UnknownVal(attr), cty.NilVal, path.GetAttr(name))
			diags = diags.Append(childDiags)
			attrs[name] = child
		}
		return cty.ObjectVal(attrs), nil
	default:
		panic(fmt.Errorf("unknown complex type: %s", original.Type().FriendlyName()))
	}
}

// DataFiller should be used by data sources to populate their computed values.
//
// As data sources skip the middle apply stage, we use the unknownFiller to
// work out if a resource should be populated and then use the valueFiller to
// actually populate it.
func DataFiller() Filler {
	return &dataFiller{
		unknown: new(unknownFiller),
		value:   new(valueFiller),
	}
}

type dataFiller struct {
	unknown *unknownFiller
	value   *valueFiller
}

func (d dataFiller) Process(original cty.Value) bool {
	return d.unknown.Process(original)
}

func (d dataFiller) Fill(original, target cty.Value, path cty.Path) (cty.Value, tfdiags.Diagnostics) {
	return d.value.Fill(original, target, path)
}

func str(n int) string {
	b := make([]rune, n)
	for i := range b {
		if testRand != nil {
			b[i] = chars[testRand.Intn(len(chars))]
		} else {
			b[i] = chars[rand.Intn(len(chars))]
		}
	}
	return string(b)
}

func getChildSafe(value cty.Value, attr string) cty.Value {
	if value == cty.NilVal {
		return cty.NilVal
	}

	if value.Type().HasAttribute(attr) {
		return value.GetAttr(attr)
	}
	return cty.NilVal
}
