/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/identifiers"
	"github.com/hashicorp/go-tfe"
	"log"
	"net/url"
	"regexp"
	"github.com/hashicorp/aws-service-catalog-engine-for-tfc/engine/lambda-functions/shared/tfc"
	"sort"
)

func FetchRunOutputs(ctx context.Context, client *tfe.Client, request NotifyRunResultRequest) ([]types.RecordOutput, error) {
	// Get workspace name
	workspaceName := identifiers.GetWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)
	w, err := client.Workspaces.Read(ctx, request.TerraformOrganization, workspaceName)
	if err != nil {
		return nil, tfc.Error(err)
	}

	run, err := client.Runs.Read(ctx, request.TerraformRunId)
	if err != nil {
		return nil, tfc.Error(err)
	}

	// Get state version of the Apply
	stateVersion, err := GetStateVersionFromRun(ctx, client, run, w)
	if err != nil {
		return nil, err
	}

	// If no state version was found, return nothing
	if stateVersion == nil {
		return nil, nil
	}

	// Get state version outputs
	log.Default().Print("Fetching run outputs from state version...")
	stateVersionOutputs, err := GetAllStateVersionOutputs(ctx, client, stateVersion.ID, 0)
	if err != nil {
		return nil, err
	}

	log.Default().Print("Mapping run outputs...")
	var recordOutputs []types.RecordOutput
	for _, stateVersionOutput := range stateVersionOutputs {
		recordOutputs = append(recordOutputs, types.RecordOutput{
			OutputKey:   aws.String(stateVersionOutput.Name),
			OutputValue: aws.String(fmt.Sprintf("%v", stateVersionOutput.Value)),
		})
	}

	// Map "State Version outputs" into "Service Catalog record outputs"
	return recordOutputs, nil
}

func GetStateVersionFromRun(ctx context.Context, client *tfe.Client, run *tfe.Run, workspace *tfe.Workspace) (*tfe.StateVersion, error) {
	// Get the Apply
	if run.Apply == nil {
		return nil, errors.New("run from TFC was missing apply data, retry again later")
	}

	applyID := run.Apply.ID

	return GetCurrentStateVersionForApply(ctx, client, applyID, workspace)
}

// Apply represents a Terraform Enterprise apply.
type Apply struct {
	ID                   string                     `jsonapi:"primary,applies"`
	LogReadURL           string                     `jsonapi:"attr,log-read-url"`
	ResourceAdditions    int                        `jsonapi:"attr,resource-additions"`
	ResourceChanges      int                        `jsonapi:"attr,resource-changes"`
	ResourceDestructions int                        `jsonapi:"attr,resource-destructions"`
	Status               tfe.ApplyStatus            `jsonapi:"attr,status"`
	StatusTimestamps     *tfe.ApplyStatusTimestamps `jsonapi:"attr,status-timestamps"`
	StateVersions        []*tfe.StateVersion        `jsonapi:"relation,state-versions,omitempty"`
}

func GetCurrentStateVersionForApply(ctx context.Context, client *tfe.Client, applyID string, workspace *tfe.Workspace) (*tfe.StateVersion, error) {
	if !validStringID(&applyID) {
		return nil, tfe.ErrInvalidApplyID
	}

	u := fmt.Sprintf("applies/%s", url.QueryEscape(applyID))
	req, err := client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	// Get the Apply
	a := &Apply{}
	err = req.Do(ctx, a)
	if err != nil {
		return nil, tfc.Error(err)
	}

	log.Default().Printf("Apply status is currently: %s", a.Status)

	log.Default().Printf("Found %d state versions for Apply", len(a.StateVersions))

	var currentStateVersion *tfe.StateVersion
	if len(a.StateVersions) == 0 {
		log.Default().Print("Falling back to fetching latest state version for workspace...")

		// If Run wasn't applied due to no changes being present in the Plan, fetch the latest State Version
		currentStateVersion, err = client.StateVersions.ReadCurrent(ctx, workspace.ID)
		if err == tfe.ErrResourceNotFound {
			log.Default().Print("No state versions found for workspace")
			return nil, nil
		}
		return currentStateVersion, tfc.Error(err)
	}

	// Fetch each of the Apply instance's State Versions
	for i, stateVersion := range a.StateVersions {
		loadedStateVersion, err := client.StateVersions.Read(ctx, stateVersion.ID)
		if err != nil {
			return nil, err
		}
		a.StateVersions[i] = loadedStateVersion
	}

	// Sort the state versions by creation date. In the current version of TFC, long Apply objects can sometimes return
	// multiple state versions due to a bug. In order to maintain compatibility with TFE, we need to handle this bug by
	// NOT relying on the fact that there should only be a single state version per Apply
	sort.Slice(a.StateVersions, func(i, j int) bool {
		return a.StateVersions[i].CreatedAt.Before(a.StateVersions[j].CreatedAt)
	})

	// Return the latest state version
	return a.StateVersions[len(a.StateVersions)-1], nil
}

func GetAllStateVersionOutputs(ctx context.Context, client *tfe.Client, stateVersionID string, pageNumber int) ([]*tfe.StateVersionOutput, error) {
	stateVersionOutputs, err := client.StateVersions.ListOutputs(ctx, stateVersionID, &tfe.StateVersionOutputsListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
	})
	if err != nil {
		return nil, tfc.Error(err)
	}

	// If more state version outputs exists, fetch them and return them as well
	if stateVersionOutputs.TotalCount > ((pageNumber + 1) * 100) {
		outputs, err := GetAllStateVersionOutputs(ctx, client, stateVersionID, pageNumber+1)
		if err != nil {
			return nil, err
		}
		return append(stateVersionOutputs.Items, outputs...), err
	} else {
		return stateVersionOutputs.Items, err
	}
}

// validStringID checks if the given string pointer is non-nil and
// contains a typical string identifier.
func validStringID(v *string) bool {
	var reStringID = regexp.MustCompile(`^[a-zA-Z0-9\-._]+$`)
	return v != nil && reStringID.MatchString(*v)
}
