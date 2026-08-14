package projectapi

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/stytchauth/stytch-go/v18/stytch/b2b/b2bstytchapi"
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
	newClient    func(projectID, secret, baseURI string) (*stytchapi.API, error)
	newB2BClient func(projectID, secret, baseURI string) (*b2bstytchapi.API, error)
	baseURI      string
	mgmtOverride bool
	mu           sync.Mutex
	projectIDs   map[string]string
	locks        map[string]*sync.Mutex
}

// LockKeyConnectedApp and LockKeyOrganization namespace Lock keys so IDs from
// different resource types cannot collide in the shared map.
func LockKeyConnectedApp(clientID string) string {
	return "connected_app/" + clientID
}

func LockKeyOrganization(organizationID string) string {
	return "organization/" + organizationID
}

type Option func(*Factory)

// WithBaseURI overrides the project API host, which otherwise derives from the
// project ID prefix.
func WithBaseURI(baseURI string) Option {
	return func(f *Factory) {
		f.baseURI = baseURI
	}
}

// WithManagementBaseURIOverridden records that the provider's base_uri points
// somewhere other than the public management API. Without a matching project
// API override, every project-level call would silently reach production, so
// ForEnvironment refuses to build a client instead.
func WithManagementBaseURIOverridden() Option {
	return func(f *Factory) {
		f.mgmtOverride = true
	}
}

func NewFactory(mgmt *api.API, opts ...Option) *Factory {
	return newFactory(managementAdapter{api: mgmt}, defaultNewClient, opts...)
}

func newFactory(mgmt ManagementAPI, newClient func(projectID, secret, baseURI string) (*stytchapi.API, error), opts ...Option) *Factory {
	f := &Factory{
		mgmt:         mgmt,
		newClient:    newClient,
		newB2BClient: defaultNewB2BClient,
		projectIDs:   map[string]string{},
		locks:        map[string]*sync.Mutex{},
	}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

// JWKS initialization is skipped because it performs a network fetch that
// connected app management never needs.
func defaultNewClient(projectID, secret, baseURI string) (*stytchapi.API, error) {
	opts := []stytchapi.Option{stytchapi.WithSkipJWKSInitialization()}
	if baseURI != "" {
		opts = append(opts, stytchapi.WithBaseURI(baseURI))
	}
	return stytchapi.NewClient(projectID, secret, opts...)
}

func defaultNewB2BClient(projectID, secret, baseURI string) (*b2bstytchapi.API, error) {
	opts := []b2bstytchapi.Option{b2bstytchapi.WithSkipJWKSInitialization()}
	if baseURI != "" {
		opts = append(opts, b2bstytchapi.WithBaseURI(baseURI))
	}
	return b2bstytchapi.NewClient(projectID, secret, opts...)
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

func (f *Factory) resolveCredentials(ctx context.Context, projectSlug, environmentSlug, secret string) (string, string, error) {
	if f.mgmtOverride && f.baseURI == "" {
		return "", "", fmt.Errorf("the provider sets base_uri but not project_api_base_uri: the project API host derives from the " +
			"project ID rather than from base_uri, so this resource would reach the public Stytch API instead of the " +
			"configured one. Set project_api_base_uri (or the STYTCH_PROJECT_API_BASE_URI environment variable)")
	}

	if secret == "" {
		secret = os.Getenv(ImportSecretEnvVar)
	}
	if secret == "" {
		return "", "", fmt.Errorf("no project secret available for %s/%s: set the project_secret attribute, or the %s environment variable when importing", projectSlug, environmentSlug, ImportSecretEnvVar)
	}

	resolvedProjectID, err := f.projectID(ctx, projectSlug, environmentSlug)
	if err != nil {
		return "", "", fmt.Errorf("resolving project ID for %s/%s: %w", projectSlug, environmentSlug, err)
	}

	return resolvedProjectID, secret, nil
}

func (f *Factory) ForEnvironment(ctx context.Context, projectSlug, environmentSlug, secret string) (*stytchapi.API, error) {
	resolvedProjectID, resolvedSecret, err := f.resolveCredentials(ctx, projectSlug, environmentSlug, secret)
	if err != nil {
		return nil, err
	}
	return f.newClient(resolvedProjectID, resolvedSecret, f.baseURI)
}

// ForB2BEnvironment builds a B2B-vertical client: B2B-only resources live on
// a different stytch-go client type than the consumer surface.
func (f *Factory) ForB2BEnvironment(ctx context.Context, projectSlug, environmentSlug, secret string) (*b2bstytchapi.API, error) {
	resolvedProjectID, resolvedSecret, err := f.resolveCredentials(ctx, projectSlug, environmentSlug, secret)
	if err != nil {
		return nil, err
	}
	return f.newB2BClient(resolvedProjectID, resolvedSecret, f.baseURI)
}

// Lock serializes writes that share a target the API cannot compare-and-swap.
// Keys from different resource types share one keyspace; callers namespace them
// with the LockKey* helpers.
func (f *Factory) Lock(key string) func() {
	f.mu.Lock()
	lock, ok := f.locks[key]
	if !ok {
		lock = &sync.Mutex{}
		f.locks[key] = lock
	}
	f.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}
