package testutil

import (
	"github.com/hashicorp/go-tfe"
	"fmt"
	"encoding/base64"
	"crypto/sha1"
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
