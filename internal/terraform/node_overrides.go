// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs"
)

// GraphNodeAttachOverride means a resource can attach and handle an entire
// overridable object. For now this is only NodeAbstractResourceInstance, but
// we include this to make extending the override functionality easier later.
type GraphNodeAttachOverride interface {
	GraphNodeResourceInstance

	AttachOverride(override *configs.ResourceOverride)
}

// GraphNodeSetOverrideValue is similar to GraphNodeAttachOverride except it attaches
// a single attribute from the matched configs.ResourceOverride, retrieved by
// a call to the GetOverrideKey function.
type GraphNodeSetOverrideValue interface {
	GraphNodeModuleInstance

	GetOverrideKey() string
	SetOverride(value cty.Value)
}
