package projectapi

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/stytchauth/stytch-go/v18/stytch/consumer/stytchapi"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	"github.com/stytchauth/stytch-management-go/v3/pkg/models/environments"
)

// Terraform sends no configuration with the import RPC, so an imported resource
// has no project_secret until the first apply writes one into state.
const ImportSecretEnvVar = "STYTCH_IMPORT_PROJECT_SECRET"

type ManagementAPI interface {
	GetProjectID(ctx context.Context, projectSlug, environmentSlug string) (string, error)
}

type managementAdapter struct {
	api *api.API
}

func (a managementAdapter) GetProjectID(ctx context.Context, projectSlug, environmentSlug string) (string, error) {
	resp, err := a.api.Environments.Get(ctx, environments.GetRequest{
		ProjectSlug:     projectSlug,
		EnvironmentSlug: environmentSlug,
	})
	if err != nil {
		return "", err
	}
	if resp.Environment.ProjectID == "" {
		return "", fmt.Errorf("environment %s/%s has no project_id", projectSlug, environmentSlug)
	}
	return resp.Environment.ProjectID, nil
}

type Factory struct {
	mgmt         ManagementAPI
	newClient    func(projectID, secret string) (*stytchapi.API, error)
	mu           sync.Mutex
	projectIDs   map[string]string
	lockByClient map[string]*sync.Mutex
}

func NewFactory(mgmt *api.API) *Factory {
	return newFactory(managementAdapter{api: mgmt}, defaultNewClient)
}

func newFactory(mgmt ManagementAPI, newClient func(string, string) (*stytchapi.API, error)) *Factory {
	return &Factory{
		mgmt:         mgmt,
		newClient:    newClient,
		projectIDs:   map[string]string{},
		lockByClient: map[string]*sync.Mutex{},
	}
}

// JWKS initialization is skipped because it performs a network fetch that
// connected app management never needs.
func defaultNewClient(projectID, secret string) (*stytchapi.API, error) {
	return stytchapi.NewClient(projectID, secret, stytchapi.WithSkipJWKSInitialization())
}

func (f *Factory) projectID(ctx context.Context, projectSlug, environmentSlug string) (string, error) {
	key := projectSlug + "/" + environmentSlug
	f.mu.Lock()
	cached, ok := f.projectIDs[key]
	f.mu.Unlock()
	if ok {
		return cached, nil
	}
	resolved, err := f.mgmt.GetProjectID(ctx, projectSlug, environmentSlug)
	if err != nil {
		return "", err
	}
	f.mu.Lock()
	f.projectIDs[key] = resolved
	f.mu.Unlock()
	return resolved, nil
}

func (f *Factory) ForEnvironment(ctx context.Context, projectSlug, environmentSlug, secret string) (*stytchapi.API, error) {
	if secret == "" {
		secret = os.Getenv(ImportSecretEnvVar)
	}
	if secret == "" {
		return nil, fmt.Errorf("no project secret available for %s/%s: set the project_secret attribute, or the %s environment variable when importing", projectSlug, environmentSlug, ImportSecretEnvVar)
	}

	resolvedProjectID, err := f.projectID(ctx, projectSlug, environmentSlug)
	if err != nil {
		return nil, fmt.Errorf("resolving project ID for %s/%s: %w", projectSlug, environmentSlug, err)
	}

	return f.newClient(resolvedProjectID, secret)
}

func (f *Factory) LockClient(clientID string) func() {
	f.mu.Lock()
	lock, ok := f.lockByClient[clientID]
	if !ok {
		lock = &sync.Mutex{}
		f.lockByClient[clientID] = lock
	}
	f.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}
