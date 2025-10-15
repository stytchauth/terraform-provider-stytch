package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stytchauth/stytch-management-go/v3/pkg/api"
	migrationprojects "github.com/stytchauth/stytch-management-go/v3/pkg/models/migration/projects"
)

const defaultLiveEnvironmentSlug = "production"

func resolveLegacyProject(
	ctx context.Context, client *api.API, projectID string,
) (*migrationprojects.LegacyProject, diag.Diagnostics) {
	var diags diag.Diagnostics

	if projectID == "" {
		diags.AddError(
			"Missing legacy project ID",
			"The stored Terraform state did not contain a project identifier, so it cannot be upgraded automatically.",
		)
		return nil, diags
	}

	resp, err := client.V1ToV3MigrationClient.GetProject(ctx, migrationprojects.GetProjectRequest{
		ProjectID: projectID,
	})
	if err != nil {
		diags.AddError(
			"Failed to retrieve legacy project metadata",
			err.Error(),
		)
		return nil, diags
	}

	project := resp.Project
	if project.ProjectSlug == "" {
		diags.AddError(
			"Missing project slug",
			fmt.Sprintf("The migration endpoint returned an empty project slug for project ID %s.", projectID),
		)
	}

	return &project, diags
}

func resolveLegacyProjectAndEnvironment(
	ctx context.Context, client *api.API, projectID string,
) (string, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	project, diags := resolveLegacyProject(ctx, client, projectID)
	if diags.HasError() {
		return "", "", diags
	}

	var environmentSlug string

	switch projectID {
	case project.LiveProjectID:
		environmentSlug = project.LiveEnvironmentSlug
		if environmentSlug == "" {
			environmentSlug = defaultLiveEnvironmentSlug
		}
	case project.TestProjectID:
		environmentSlug = project.TestEnvironmentSlug
	default:
		diags.AddError(
			"Unknown legacy project identifier",
			fmt.Sprintf("Project ID %s did not match the live or test project associated with slug %s.", projectID, project.ProjectSlug),
		)
		return "", "", diags
	}

	if environmentSlug == "" {
		diags.AddError(
			"Missing environment slug",
			fmt.Sprintf("No environment slug could be determined for project ID %s.", projectID),
		)
		return "", "", diags
	}

	return project.ProjectSlug, environmentSlug, diags
}
