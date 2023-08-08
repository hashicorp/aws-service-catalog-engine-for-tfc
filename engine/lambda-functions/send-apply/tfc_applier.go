/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tfc"
	"github.com/hashicorp/go-tfe"
	"log"
	"net/http"
)

const ProviderAuthVariableKey = "TFC_AWS_PROVIDER_AUTH"
const RunRoleArnVariableKey = "TFC_AWS_RUN_ROLE_ARN"

const ProductIdMetadataHeaderKey = "Tfp-Aws-Service-Catalog-Product-Id"
const ProvisionedProductIdMetadataHeaderKey = "Tfp-Aws-Service-Catalog-Prv-Product-Id"
const ProductVersionMetadataHeaderKey = "Tfp-Aws-Service-Catalog-Product-Ver"

type TFCApplier struct {
	tfeClient        *tfe.Client
	terraformVersion string
}

func (h *SendApplyHandler) NewTFCApplier(ctx context.Context, request SendApplyRequest) (*TFCApplier, error) {
	headers := http.Header{}

	headers.Set(ProductIdMetadataHeaderKey, request.ProductId)
	headers.Set(ProvisionedProductIdMetadataHeaderKey, request.ProvisionedProductId)
	headers.Set(ProductVersionMetadataHeaderKey, request.ProvisionedArtifactId)

	tfeClient, err := tfc.GetTFEClientWithHeaders(ctx, h.secretsManager, headers)
	return &TFCApplier{
		tfeClient:        tfeClient,
		terraformVersion: h.terraformVersion,
	}, err
}

func (applier *TFCApplier) FindOrCreateProject(ctx context.Context, organizationName string, name string) (*tfe.Project, error) {
	log.Default().Printf("finding or creating TFC project with name: %s", name)

	// Check if the project already exists...
	project, err := applier.FindProjectByName(ctx, organizationName, name, 0)
	if project != nil || err != nil {
		if err == nil {
			log.Default().Printf("found existing project with id: %s", project.ID)
		}
		return project, err
	}

	// Otherwise, create the project
	log.Default().Printf("no existing project found, creating new project...")
	newProject, err := applier.tfeClient.Projects.Create(ctx, organizationName, tfe.ProjectCreateOptions{
		Name: name,
	})
	return newProject, tfc.Error(err)
}

func (applier *TFCApplier) FindProjectByName(ctx context.Context, organizationName string, projectName string, pageNumber int) (*tfe.Project, error) {
	// Check if the project already exists...
	projects, err := applier.tfeClient.Projects.List(ctx, organizationName, &tfe.ProjectListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
		Name: projectName,
	})
	if err != nil {
		return nil, tfc.Error(err)
	}

	for _, project := range projects.Items {
		// Check for exact name match, because the search we made is a "contains" search
		if project.Name == projectName {
			return project, nil
		}
	}

	// If more projects exists, fetch them and check them as well
	if projects.TotalCount > ((pageNumber + 1) * 100) {
		return applier.FindProjectByName(ctx, organizationName, projectName, pageNumber+1)
	}

	return nil, nil
}

func (applier *TFCApplier) FindOrCreateWorkspace(ctx context.Context, organizationName string, project *tfe.Project, workspaceName string) (*tfe.Workspace, error) {
	// Check if the workspace already exists...
	workspace, err := applier.FindWorkspaceByName(ctx, organizationName, workspaceName, 0)
	if workspace != nil || err != nil {
		if err == nil {
			log.Default().Printf("found existing workspace with id: %s", workspace.ID)
		}
		return workspace, err
	}

	// Otherwise, create the Workspace
	log.Default().Printf("no existing workspace found, creating new workspace...")
	newWorkspace, err := applier.tfeClient.Workspaces.Create(ctx, organizationName, tfe.WorkspaceCreateOptions{
		Name:    tfe.String(workspaceName),
		Project: project,
	})
	return newWorkspace, tfc.Error(err)
}

func (applier *TFCApplier) FindWorkspaceByName(ctx context.Context, organizationName string, workspaceName string, pageNumber int) (*tfe.Workspace, error) {
	// Check if the workspace already exists...
	workspaces, err := applier.tfeClient.Workspaces.List(ctx, organizationName, &tfe.WorkspaceListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
		Search: workspaceName,
	})
	if err != nil {
		return nil, tfc.Error(err)
	}

	for _, workspace := range workspaces.Items {
		// Check for exact name match, because the search we made is a "contains" search
		if workspace.Name == workspaceName {
			return workspace, nil
		}
	}

	// If more workspaces exists, fetch them and check them as well
	if workspaces.TotalCount > ((pageNumber + 1) * 100) {
		return applier.FindWorkspaceByName(ctx, organizationName, workspaceName, pageNumber+1)
	}

	return nil, nil
}

func (applier *TFCApplier) UpdateWorkspaceTerraformVersion(ctx context.Context, workspaceId string) error {
	log.Default().Printf("Setting terraform version of %s to %s", workspaceId, applier.terraformVersion)
	_, err := applier.tfeClient.Workspaces.UpdateByID(ctx, workspaceId, tfe.WorkspaceUpdateOptions{
		TerraformVersion: tfe.String(applier.terraformVersion),
	})
	return tfc.Error(err)
}

func (applier *TFCApplier) UpdateWorkspaceOIDCVariables(ctx context.Context, w *tfe.Workspace, launchRoleArn string) error {
	log.Default().Printf("Updating OIDC variables")
	err := applier.FindOrCreateENVVariable(ctx, w, ProviderAuthVariableKey, "true", "Enable the Workload Identity integration for AWS.")
	if err != nil {
		return err
	}

	return applier.FindOrCreateENVVariable(ctx, w, RunRoleArnVariableKey, launchRoleArn, "The AWS role ARN runs will use to authenticate.")
}

func (applier *TFCApplier) UpdateWorkspaceParameterVariables(ctx context.Context, w *tfe.Workspace, parameters []Parameter) error {
	for _, parameter := range parameters {
		log.Default().Printf("Updating variable %s", parameter.Key)
		err := applier.FindOrCreateTerraformVariable(ctx, w, parameter.Key, parameter.Value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (applier *TFCApplier) CreateConfigurationVersion(ctx context.Context, workspaceId string) (*tfe.ConfigurationVersion, error) {
	newConfigurationVersion, err := applier.tfeClient.ConfigurationVersions.Create(ctx,
		workspaceId,
		tfe.ConfigurationVersionCreateOptions{
			// Disable auto queue runs, so we can create the run ourselves to get the runId
			AutoQueueRuns: tfe.Bool(false),
		},
	)
	return newConfigurationVersion, tfc.Error(err)
}

func (applier *TFCApplier) FindOrCreateTerraformVariable(ctx context.Context, w *tfe.Workspace, key string, value string) error {
	return applier.findOrCreateVariable(ctx, w, key, value, tfe.CategoryTerraform, "Provided via AWS Service Catalog")
}

func (applier *TFCApplier) FindOrCreateENVVariable(ctx context.Context, w *tfe.Workspace, key string, value string, description string) error {
	return applier.findOrCreateVariable(ctx, w, key, value, tfe.CategoryEnv, description)
}

func (applier *TFCApplier) findOrCreateVariable(ctx context.Context, w *tfe.Workspace, key string, value string, category tfe.CategoryType, description string) error {
	variableToUpdate, err := applier.findVariableByKey(ctx, w, key, 0)
	if err != nil {
		return err
	}

	if variableToUpdate != nil {
		// Update the variables
		log.Default().Printf("Updating variable for %s with ID: %s", key, variableToUpdate.ID)
		_, err = applier.tfeClient.Variables.Update(ctx, w.ID, variableToUpdate.ID, tfe.VariableUpdateOptions{
			Key:      tfe.String(key),
			Value:    tfe.String(value),
			Category: tfe.Category(category),
			HCL:      tfe.Bool(false),
		})
		return tfc.Error(err)
	}

	// Create the variable as it does not currently exist
	log.Default().Printf("Creating variable for %s", key)
	_, err = applier.tfeClient.Variables.Create(ctx, w.ID, tfe.VariableCreateOptions{
		Key:         tfe.String(key),
		Value:       tfe.String(value),
		Description: tfe.String(description),
		Category:    tfe.Category(category),
		HCL:         tfe.Bool(false),
		Sensitive:   tfe.Bool(false),
	})
	return tfc.Error(err)
}

func (applier *TFCApplier) findVariableByKey(ctx context.Context, w *tfe.Workspace, key string, pageNumber int) (*tfe.Variable, error) {
	variables, err := applier.tfeClient.Variables.List(ctx, w.ID, &tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
	})
	if err != nil {
		return nil, tfc.Error(err)
	}

	for _, variable := range variables.Items {
		if variable.Key == key {
			return variable, nil
		}
	}

	// If more variables exists, fetch them and check them as well
	if variables.TotalCount > ((pageNumber + 1) * 100) {
		return applier.findVariableByKey(ctx, w, key, pageNumber+1)
	}

	return nil, nil
}

// PurgeVariables purges all non-recognized variables from the workspace. This helps ensure parity between Service Catalog and TFC
func (applier *TFCApplier) PurgeVariables(ctx context.Context, w *tfe.Workspace, parameters []Parameter) error {
	log.Default().Printf("building lookups for unrecognized variables in workspace")
	allowedTerraformVarKeysMap := map[string]bool{}
	for _, parameter := range parameters {
		allowedTerraformVarKeysMap[parameter.Key] = true
	}

	allowedEnvVarKeysMap := map[string]struct{}{
		ProviderAuthVariableKey: {},
		RunRoleArnVariableKey:   {},
	}

	// Collect the entire list of variables that should be purged
	log.Default().Printf("checking for unrecognized variables in workspace")
	variablesToPurge, err := applier.checkVariablesForPurge(ctx, w, allowedTerraformVarKeysMap, allowedEnvVarKeysMap, []*tfe.Variable{}, 0)
	if err != nil {
		return err
	}

	// Purge the variables
	for _, variablesToPurge := range variablesToPurge {
		err := applier.tfeClient.Variables.Delete(ctx, w.ID, variablesToPurge.ID)
		if err != nil {
			return tfc.Error(err)
		}
	}

	return nil
}

func (applier *TFCApplier) checkVariablesForPurge(ctx context.Context, w *tfe.Workspace, allowedTerraformVarKeysMap map[string]bool, allowedEnvVarKeysMap map[string]struct{}, variablesToPurge []*tfe.Variable, pageNumber int) ([]*tfe.Variable, error) {
	variables, err := applier.tfeClient.Variables.List(ctx, w.ID, &tfe.VariableListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
	})
	if err != nil {
		return nil, tfc.Error(err)
	}

	for _, variable := range variables.Items {
		// Check if the variable is in the appropriate allowlist
		if variable.Category == tfe.CategoryTerraform {
			_, found := allowedTerraformVarKeysMap[variable.Key]
			if found {
				continue
			}
			log.Default().Printf("Terraform variable %s is being removed from workspace since it is outside the parameters list", variable.Key)
		} else if variable.Category == tfe.CategoryEnv {
			_, found := allowedEnvVarKeysMap[variable.Key]
			if found {
				continue
			}
			log.Default().Printf("ENV variable %s is being removed from workspace because it is not recognized by the engine", variable.Key)
		}

		// collect the variable so it can be deleted later (after we have finished paginating through all variables)
		variablesToPurge = append(variablesToPurge, variable)
	}

	// If more variables exists, check them as well
	if variables.TotalCount > ((pageNumber + 1) * 100) {
		return applier.checkVariablesForPurge(ctx, w, allowedTerraformVarKeysMap, allowedEnvVarKeysMap, variablesToPurge, pageNumber+1)
	}

	return variablesToPurge, nil
}
