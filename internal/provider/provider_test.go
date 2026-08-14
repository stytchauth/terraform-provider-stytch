package provider

import (
	"context"
	"testing"
)

func TestExperimentalRegistrationsRequireEnvVar(t *testing.T) {
	p := &StytchProvider{}

	t.Setenv(ExperimentalEnvVar, "")
	baseResources := len(p.Resources(context.Background()))
	baseDataSources := len(p.DataSources(context.Background()))

	t.Setenv(ExperimentalEnvVar, "1")
	if got := len(p.Resources(context.Background())); got != baseResources+1 {
		t.Fatalf("expected %d resources with experimental enabled, got %d", baseResources+1, got)
	}
	if got := len(p.DataSources(context.Background())); got != baseDataSources+1 {
		t.Fatalf("expected %d data sources with experimental enabled, got %d", baseDataSources+1, got)
	}

	// Only the exact value 1 enables the gate.
	t.Setenv(ExperimentalEnvVar, "true")
	if got := len(p.Resources(context.Background())); got != baseResources {
		t.Fatalf("expected the gate to require the value 1, got %d resources", got)
	}
}
