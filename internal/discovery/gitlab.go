package discovery

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gitlab.com/gitlab-org/api/client-go"
)

// FetchGitLabConfig retrieves and parses a JSON config file from a GitLab repository
func FetchGitLabConfig(token, baseURL, projectID, filePath, ref string) (Config, error) {
	var conf Config

	// Initialize GitLab Client
	gl, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return conf, fmt.Errorf("failed to create gitlab client: %w", err)
	}

	// Fetch the file
	file, _, err := gl.RepositoryFiles.GetFile(projectID, filePath, &gitlab.GetFileOptions{Ref: &ref})
	if err != nil {
		return conf, fmt.Errorf("failed to fetch file from gitlab: %w", err)
	}

	// GitLab API returns content as Base64 encoded string
	decoded, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return conf, fmt.Errorf("failed to decode gitlab file content: %w", err)
	}

	// Unmarshal the JSON content into our Config struct
	if err := json.Unmarshal(decoded, &conf); err != nil {
		return conf, fmt.Errorf("failed to parse json config: %w", err)
	}

	return conf, nil
}
