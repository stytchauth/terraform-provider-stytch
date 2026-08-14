package resources

import "fmt"

// ExperimentalEnvVar enables experimental resources and data sources when set
// to 1. They stay registered either way so use without the variable fails with
// the instructive diagnostic below rather than Terraform's generic "provider
// does not support resource type" - including for instances already in state.
const ExperimentalEnvVar = "STYTCH_PROVIDER_USE_EXPERIMENTAL_RESOURCES"

func experimentalGateError(typeName string) (summary, detail string) {
	return "Experimental feature not enabled",
		fmt.Sprintf("%s is experimental and currently disabled. Set %s=1 in the environment where Terraform runs "+
			"(including CI) to enable it; every operation on it - plan, apply, refresh, import, and destroy - requires "+
			"the variable. Experimental schemas may change in a future release without a major version bump.",
			typeName, ExperimentalEnvVar)
}
