package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/hashicorp/go-tfe"
	"log"
	"net/url"
	"regexp"
	"github.com/hashicorp/aws-service-catalog-enginer-for-tfe/lambda-functions/golang/shared/identifiers"
)

func FetchRunOutputs(ctx context.Context, client *tfe.Client, request NotifyRunResultRequest) ([]types.RecordOutput, error) {
	// Get workspace name
	workspaceName := identifiers.GetWorkspaceName(request.AwsAccountId, request.ProvisionedProductId)
	w, err := client.Workspaces.Read(ctx, request.TerraformOrganization, workspaceName)
	if err != nil {
		return nil, err
	}

	run, err := client.Runs.Read(ctx, request.TerraformRunId)
	if err != nil {
		return nil, err
	}

	// Get state version of the Apply
	stateVersion, err := GetStateVersionFromRun(ctx, client, run, w)
	if err != nil {
		return nil, err
	}

	// Get state version outputs
	stateVersionOutputs, err := GetAllStateVersionOutputs(ctx, client, stateVersion.ID, 0)
	if err != nil {
		return nil, err
	}

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
		return nil, err
	}

	log.Default().Printf("Apply status is currently: %s", a.Status)

	// We expect there will be only one state version for the Apply. It is a has-many relationship due to
	// legacy decisions, but all modern versions of Terraform should only have a single State Version.
	if len(a.StateVersions) > 1 {
		return nil, errors.New("too many state versions exist for this run to determine the current state version")
	}

	var currentStateVersion *tfe.StateVersion
	if len(a.StateVersions) == 0 {
		// If Run wasn't applied due to no changes being present in the Plan, fetch the latest State Version
		currentStateVersion, err = client.StateVersions.ReadCurrent(ctx, workspace.ID)
		if err != nil {
			return nil, err
		}

		if currentStateVersion == nil {
			return nil, errors.New("run has no state version")
		}
	} else {
		currentStateVersion = a.StateVersions[0]
	}

	return currentStateVersion, nil
}

func GetAllStateVersionOutputs(ctx context.Context, client *tfe.Client, stateVersionID string, pageNumber int) ([]*tfe.StateVersionOutput, error) {
	stateVersionOutputs, err := client.StateVersions.ListOutputs(ctx, stateVersionID, &tfe.StateVersionOutputsListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: pageNumber,
			PageSize:   100,
		},
	})
	if err != nil {
		return nil, err
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
