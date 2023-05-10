package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/servicecatalog/types"
	"github.com/hashicorp/go-tfe"
	"net/url"
	"regexp"
)

func FetchRunOutputs(ctx context.Context, client *tfe.Client, terraformRunId string) ([]types.RecordOutput, error) {
	run, err := client.Runs.Read(ctx, terraformRunId)
	if err != nil {
		return nil, err
	}

	// Get state version of the apply
	stateVersion, err := GetStateVersionFromRun(ctx, client, run.ID)
	if err != nil {
		return nil, err
	}

	// Get state version outputs
	stateVersionOutputs, err := GetAllStateVersionOutputs(ctx, client, stateVersion.ID)

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

func GetStateVersionFromRun(ctx context.Context, client *tfe.Client, runId string) (*tfe.StateVersion, error) {
	run, err := client.Runs.Read(ctx, runId)
	if err != nil {
		return nil, err
	}

	// Get the apply
	if run.Apply == nil {
		return nil, errors.New("run from TFC was missing apply data, retry again later")
	}

	applyID := run.Apply.ID

	return GetCurrentStateVersionForApply(ctx, client, applyID)
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

func GetCurrentStateVersionForApply(ctx context.Context, client *tfe.Client, applyID string) (*tfe.StateVersion, error) {
	if !validStringID(&applyID) {
		return nil, tfe.ErrInvalidApplyID
	}

	u := fmt.Sprintf("applies/%s", url.QueryEscape(applyID))
	req, err := client.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	// Get apply
	a := &Apply{}
	err = req.Do(ctx, a)
	if err != nil {
		return nil, err
	}

	// We expect there will be only one state version for the Apply. It is a has-many relationship due to
	// legacy decisions, but all modern versions of Terraform should only have a single State Version.
	if len(a.StateVersions) > 1 {
		return nil, errors.New("too many state versions exist for this run to determine the current state version")
	}

	if len(a.StateVersions) == 0 {
		return nil, errors.New("run has no state version")
	}

	currentStateVersion := a.StateVersions[0]

	return currentStateVersion, nil
}

func GetAllStateVersionOutputs(ctx context.Context, client *tfe.Client, stateVersionID string) ([]*tfe.StateVersionOutput, error) {
	//TODO: Paginate through state version outputs to make sure we don't miss any
	stateVersionOutputs, err := client.StateVersions.ListOutputs(ctx, stateVersionID, &tfe.StateVersionOutputsListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 0,
			PageSize:   100,
		},
	})

	return stateVersionOutputs.Items, err
}

// validStringID checks if the given string pointer is non-nil and
// contains a typical string identifier.
func validStringID(v *string) bool {
	var reStringID = regexp.MustCompile(`^[a-zA-Z0-9\-._]+$`)
	return v != nil && reStringID.MatchString(*v)
}