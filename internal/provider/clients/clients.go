package clients

import (
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/terraform-provider-stytch/internal/provider/projectapi"
)

type Clients struct {
	Management *api.API
	ProjectAPI *projectapi.Factory
	// ExperimentalEnabled reports whether experimental resources and data
	// sources may be used; they refuse to operate when it is false.
	ExperimentalEnabled bool
}
