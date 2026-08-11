package projectapi

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stytchauth/stytch-go/v18/stytch/consumer/stytchapi"
)

type fakeManagement struct {
	mu       sync.Mutex
	getCalls int
}

func (f *fakeManagement) GetProjectID(_ context.Context, projectSlug, environmentSlug string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.getCalls++
	return "project-test-" + projectSlug + "-" + environmentSlug, nil
}

type capturedClient struct {
	projectID string
	secret    string
}

func newTestFactory(mgmt ManagementAPI) (*Factory, *[]capturedClient) {
	var captured []capturedClient
	f := newFactory(mgmt, func(projectID, secret string) (*stytchapi.API, error) {
		captured = append(captured, capturedClient{projectID: projectID, secret: secret})
		return &stytchapi.API{}, nil
	})
	return f, &captured
}

func TestForEnvironmentUsesConfiguredSecret(t *testing.T) {
	mgmt := &fakeManagement{}
	f, captured := newTestFactory(mgmt)

	if _, err := f.ForEnvironment(context.Background(), "proj", "env", "my-secret"); err != nil {
		t.Fatal(err)
	}
	if (*captured)[0].projectID != "project-test-proj-env" || (*captured)[0].secret != "my-secret" {
		t.Fatalf("client built with wrong credentials: %+v", (*captured)[0])
	}
}

func TestForEnvironmentFallsBackToImportEnvVar(t *testing.T) {
	t.Setenv(ImportSecretEnvVar, "env-secret")
	f, captured := newTestFactory(&fakeManagement{})

	if _, err := f.ForEnvironment(context.Background(), "proj", "env", ""); err != nil {
		t.Fatal(err)
	}
	if (*captured)[0].secret != "env-secret" {
		t.Fatalf("expected the environment variable secret, got %q", (*captured)[0].secret)
	}
}

func TestForEnvironmentPrefersConfiguredSecretOverImportEnvVar(t *testing.T) {
	t.Setenv(ImportSecretEnvVar, "env-secret")
	f, captured := newTestFactory(&fakeManagement{})

	if _, err := f.ForEnvironment(context.Background(), "proj", "env", "my-secret"); err != nil {
		t.Fatal(err)
	}
	if (*captured)[0].secret != "my-secret" {
		t.Fatalf("expected the configured secret to win, got %q", (*captured)[0].secret)
	}
}

func TestForEnvironmentEmptySecretErrors(t *testing.T) {
	t.Setenv(ImportSecretEnvVar, "")
	mgmt := &fakeManagement{}
	f, captured := newTestFactory(mgmt)

	_, err := f.ForEnvironment(context.Background(), "proj", "env", "")
	if err == nil {
		t.Fatal("expected an error when no project secret is available")
	}
	if !strings.Contains(err.Error(), "project_secret") || !strings.Contains(err.Error(), ImportSecretEnvVar) {
		t.Fatalf("error must name both the attribute and the environment variable, got %q", err)
	}
	if len(*captured) != 0 {
		t.Fatalf("expected no client to be built, got %+v", *captured)
	}
	if mgmt.getCalls != 0 {
		t.Fatalf("expected no environment lookup, got %d", mgmt.getCalls)
	}
}

func TestForEnvironmentCachesProjectID(t *testing.T) {
	mgmt := &fakeManagement{}
	f, _ := newTestFactory(mgmt)

	for range 3 {
		if _, err := f.ForEnvironment(context.Background(), "proj", "env", "s"); err != nil {
			t.Fatal(err)
		}
	}
	if mgmt.getCalls != 1 {
		t.Fatalf("expected 1 environment lookup, got %d", mgmt.getCalls)
	}
}

func TestLockClientSerializesSameKey(t *testing.T) {
	f, _ := newTestFactory(&fakeManagement{})

	unlock := f.LockClient("client-a")
	otherUnlock := f.LockClient("client-b")
	otherUnlock()
	unlock()

	held := f.LockClient("client-a")
	started := make(chan struct{})
	acquired := make(chan struct{})
	go func() {
		close(started)
		u := f.LockClient("client-a")
		close(acquired)
		u()
	}()

	<-started
	select {
	case <-acquired:
		t.Fatal("second acquisition of client-a succeeded while the key was still held")
	case <-time.After(50 * time.Millisecond):
	}

	held()
	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("second acquisition of client-a stayed blocked after unlock")
	}
}
