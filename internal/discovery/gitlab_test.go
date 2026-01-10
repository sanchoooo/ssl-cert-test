package discovery

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andre/ssl-cert-test/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFetchGitLabConfig(t *testing.T) {
	// 1. Prepare Test Data
	expectedConfig := config.Config{
		Domains: []string{"example.com", "test.com"},
		Ports:   []int{443, 8443},
	}
	jsonBytes, _ := json.Marshal(expectedConfig)
	// GitLab API returns content Base64 encoded
	encodedContent := base64.StdEncoding.EncodeToString(jsonBytes)

	// 2. Setup Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Authentication Header
		if r.Header.Get("Private-Token") != "test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify Query Param (Ref/Branch)
		if r.URL.Query().Get("ref") != "main" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "Wrong ref")
			return
		}

		// Verify Path contains the project ID and file path
		// Note: gitlab-go library typically hits /api/v4/projects/{id}/repository/files/{path}
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Success Response Structure
		// Mimics the actual GitLab GetFile API response
		resp := map[string]interface{}{
			"file_name": "config.json",
			"file_path": "config.json",
			"size":      len(jsonBytes),
			"encoding":  "base64",
			"content":   encodedContent,
			"ref":       "main",
			"blob_id":   "1234567890",
			"commit_id": "abcdef12345",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	// 3. Execute Test
	// We pass ts.URL as the baseURL to redirect traffic to our mock
	conf, err := FetchGitLabConfig("test-token", ts.URL, "123", "config.json", "main")

	// 4. Assertions
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig.Domains, conf.Domains)
	assert.Equal(t, expectedConfig.Ports, conf.Ports)
}

func TestFetchGitLabConfig_Error_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := FetchGitLabConfig("token", ts.URL, "123", "missing.json", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch file")
}

func TestFetchGitLabConfig_Error_BadBase64(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"content": "Not_Base_64_!!!",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	_, err := FetchGitLabConfig("token", ts.URL, "123", "bad.json", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode gitlab file content")
}

func TestFetchGitLabConfig_Error_BadJSON(t *testing.T) {
	// Encode garbage data as base64 (valid base64, invalid JSON)
	garbage := "this is not json"
	encoded := base64.StdEncoding.EncodeToString([]byte(garbage))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"content": encoded,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	_, err := FetchGitLabConfig("token", ts.URL, "123", "bad.json", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse json")
}
