package projectapi

import (
	"context"
	"strings"
	"testing"

	"github.com/stytchauth/stytch-go/v18/stytch/b2b/b2bstytchapi"
	"github.com/stytchauth/stytch-go/v18/stytch/consumer/stytchapi"
)

func newTestB2BFactory(mgmt ManagementAPI, opts ...Option) (*Factory, *[]capturedClient) {
	var captured []capturedClient
	f := newFactory(mgmt, func(projectID, secret, baseURI string) (*stytchapi.API, error) {
		return &stytchapi.API{}, nil
	}, opts...)
	f.newB2BClient = func(projectID, secret, baseURI string) (*b2bstytchapi.API, error) {
		captured = append(captured, capturedClient{projectID: projectID, secret: secret, baseURI: baseURI})
		return &b2bstytchapi.API{}, nil
	}
	return f, &captured
}

func TestForB2BEnvironmentUsesConfiguredSecret(t *testing.T) {
	mgmt := &fakeManagement{}
	f, captured := newTestB2BFactory(mgmt)

	if _, err := f.ForB2BEnvironment(context.Background(), "proj", "env", "my-secret"); err != nil {
		t.Fatal(err)
	}
	if (*captured)[0].projectID != "project-test-proj-env" || (*captured)[0].secret != "my-secret" {
		t.Fatalf("client built with wrong credentials: %+v", (*captured)[0])
	}
}

func TestForB2BEnvironmentSharesProjectIDCache(t *testing.T) {
	mgmt := &fakeManagement{}
	f, _ := newTestB2BFactory(mgmt)

	if _, err := f.ForEnvironment(context.Background(), "proj", "env", "s"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.ForB2BEnvironment(context.Background(), "proj", "env", "s"); err != nil {
		t.Fatal(err)
	}
	if mgmt.getCalls != 1 {
		t.Fatalf("expected the B2B client to reuse the cached project ID, got %d lookups", mgmt.getCalls)
	}
}

func TestForB2BEnvironmentEmptySecretErrors(t *testing.T) {
	t.Setenv(ImportSecretEnvVar, "")
	f, captured := newTestB2BFactory(&fakeManagement{})

	_, err := f.ForB2BEnvironment(context.Background(), "proj", "env", "")
	if err == nil {
		t.Fatal("expected an error when no project secret is available")
	}
	if !strings.Contains(err.Error(), "project_secret") || !strings.Contains(err.Error(), ImportSecretEnvVar) {
		t.Fatalf("error must name both the attribute and the environment variable, got %q", err)
	}
	if len(*captured) != 0 {
		t.Fatalf("expected no client to be built, got %+v", *captured)
	}
}

func TestForB2BEnvironmentRequiresProjectBaseURIWhenManagementOverridden(t *testing.T) {
	f, captured := newTestB2BFactory(&fakeManagement{}, WithManagementBaseURIOverridden())

	_, err := f.ForB2BEnvironment(context.Background(), "proj", "env", "s")
	if err == nil {
		t.Fatal("expected an error when base_uri is overridden without project_api_base_uri")
	}
	if !strings.Contains(err.Error(), "project_api_base_uri") {
		t.Fatalf("error must name project_api_base_uri, got %q", err)
	}
	if len(*captured) != 0 {
		t.Fatalf("expected no client to be built, got %+v", *captured)
	}
}
