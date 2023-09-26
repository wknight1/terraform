// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package mocks

import (
	"math/rand"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
)

var (
	normalAttributes = map[string]*configschema.Attribute{
		"id": {
			Type: cty.String,
		},
		"value": {
			Type: cty.String,
		},
	}

	computedAttributes = map[string]*configschema.Attribute{
		"id": {
			Type:     cty.String,
			Computed: true,
		},
		"value": {
			Type: cty.String,
		},
	}

	normalBlock = configschema.Block{
		Attributes: normalAttributes,
	}

	computedBlock = configschema.Block{
		Attributes: computedAttributes,
	}
)

func TestFillComputedValues(t *testing.T) {

	tcs := map[string]struct {
		original cty.Value
		target   cty.Value
		schema   *configschema.Block
		filler   Filler
		expected cty.Value
	}{
		"nil_target_no_unknowns": {
			original: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
			target: cty.NilVal,
			schema: &normalBlock,
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"empty_target_no_unknowns": {
			original: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
			target: cty.EmptyObjectVal,
			schema: &normalBlock,
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"basic_computed_attribute_preset": {
			original: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
			target: cty.NilVal,
			schema: &computedBlock,
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("kj87eb9"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"basic_computed_attribute_random": {
			original: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"value": cty.StringVal("Hello, world!"),
			}),
			target: cty.NilVal,
			schema: &computedBlock,
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("ssnk9qhr"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"basic_computed_attribute_supplied": {
			original: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.UnknownVal(cty.String),
				"value": cty.StringVal("Hello, world!"),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"id": cty.StringVal("myvalue"),
			}),
			schema: &computedBlock,
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"id":    cty.StringVal("myvalue"),
				"value": cty.StringVal("Hello, world!"),
			}),
		},
		"nested_single_block_preset": {
			original: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.UnknownVal(cty.String),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
			target: cty.NilVal,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingSingle,
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("ssnk9qhr"),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
		},
		"nested_single_block_supplied": {
			original: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.UnknownVal(cty.String),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingSingle,
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("myvalue"),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
		},
		"nested_list_block_preset": {
			original: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.NilVal,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingList,
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("ssnk9qhr"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("amyllmyg"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_list_block_supplied": {
			original: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingList,
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_set_block_preset": {
			original: cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.NilVal,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingSet,
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("ssnk9qhr"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("amyllmyg"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_set_block_supplied": {
			original: cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingSet,
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_map_block_preset": {
			original: cty.ObjectVal(map[string]cty.Value{
				"block": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.NilVal,
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingMap,
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("ssnk9qhr"),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("amyllmyg"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_map_block_supplied": {
			original: cty.ObjectVal(map[string]cty.Value{
				"block": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"block": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				BlockTypes: map[string]*configschema.NestedBlock{
					"block": {
						Block:   computedBlock,
						Nesting: configschema.NestingMap,
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"block": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_single_attribute": {
			original: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.UnknownVal(cty.String),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingSingle,
						},
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("myvalue"),
					"value": cty.StringVal("Hello, world!"),
				}),
			}),
		},
		"nested_list_attribute": {
			original: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingList,
						},
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ListVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_set_attribute": {
			original: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingSet,
						},
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.SetVal([]cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
		"nested_map_attribute": {
			original: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.UnknownVal(cty.String),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
			target: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("myvalue"),
				}),
			}),
			schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"nested": {
						NestedType: &configschema.Object{
							Attributes: computedAttributes,
							Nesting:    configschema.NestingMap,
						},
					},
				},
			},
			filler: ValueFiller(),
			expected: cty.ObjectVal(map[string]cty.Value{
				"nested": cty.MapVal(map[string]cty.Value{
					"one": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("one"),
					}),
					"two": cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("myvalue"),
						"value": cty.StringVal("two"),
					}),
				}),
			}),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			// We'll just make sure that any random strings are deterministic.
			testRand = rand.New(rand.NewSource(0))
			defer func() {
				testRand = nil
			}()

			actual, err := FillComputedValues(tc.original, tc.target, tc.schema, tc.filler)
			if err != nil {
				t.Fatalf("expected no error but found %v", err)
			}

			if actual.Equals(tc.expected).False() {
				t.Fatalf("\nexpected: (%s)\nactual:   (%s)", tc.expected.GoString(), actual.GoString())
			}
		})
	}
}
