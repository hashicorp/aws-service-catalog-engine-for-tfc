/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

package testtfc

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
	"encoding/base64"
	"crypto/sha1"
	"github.com/google/uuid"
)

func WorkspaceId(workspace *tfe.Workspace) string {
	hasher := sha1.New()
	hasher.Write([]byte(workspace.Name))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	trimmedSha := TruncateString(sha, 16)
	return fmt.Sprintf("ws-%s", trimmedSha)
}

func VarId(variable *tfe.Variable) string {
	uniqueIdentifier := fmt.Sprintf("%s %s", variable.Workspace.ID, variable.Key)

	hasher := sha1.New()
	hasher.Write([]byte(uniqueIdentifier))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	trimmedSha := TruncateString(sha, 16)
	return fmt.Sprintf("var-%s", trimmedSha)
}

func ConfigVersionId(workspaceId string) string {
	uniqueIdentifier := fmt.Sprintf("%s %s", workspaceId, uuid.New().String())

	hasher := sha1.New()
	hasher.Write([]byte(uniqueIdentifier))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	trimmedSha := TruncateString(sha, 16)
	return fmt.Sprintf("cv-%s", trimmedSha)
}

func StateVersionId(workspaceId string) string {
	uniqueIdentifier := fmt.Sprintf("%s %s", workspaceId, uuid.New().String())

	hasher := sha1.New()
	hasher.Write([]byte(uniqueIdentifier))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	trimmedSha := TruncateString(sha, 16)
	return fmt.Sprintf("sv-%s", trimmedSha)
}

func RunId(run *tfe.Run) string {
	uniqueIdentifier := fmt.Sprintf("%s %s", run.Workspace.ID, uuid.New().String())

	hasher := sha1.New()
	hasher.Write([]byte(uniqueIdentifier))
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

	trimmedSha := TruncateString(sha, 16)
	return fmt.Sprintf("cv-%s", trimmedSha)
}

func TruncateString(str string, length int) string {
	if length <= 0 {
		return ""
	}

	orgLen := len(str)
	if orgLen <= length {
		return str
	}
	return str[:length]
}
