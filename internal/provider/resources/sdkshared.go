package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type sdkSMSAutofillMetadata struct {
	MetadataType  types.String `tfsdk:"metadata_type"`
	MetadataValue types.String `tfsdk:"metadata_value"`
	BundleID      types.String `tfsdk:"bundle_id"`
}
