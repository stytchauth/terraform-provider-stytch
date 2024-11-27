// Copyright (c) HashiCorp, Inc.

package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var nilSDKObject = "The object returned from the API was nil. This should never happen. Please report this issue to the provider developers."

type sdkSMSAutofillMetadata struct {
	MetadataType  types.String `tfsdk:"metadata_type"`
	MetadataValue types.String `tfsdk:"metadata_value"`
	BundleID      types.String `tfsdk:"bundle_id"`
}

func (m sdkSMSAutofillMetadata) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"metadata_type":  types.StringType,
		"metadata_value": types.StringType,
		"bundle_id":      types.StringType,
	}
}
