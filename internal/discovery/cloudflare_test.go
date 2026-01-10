package test

import (
	"encoding/json"
	"github.com/andre/ssl-cert-test/internal/config"
	"github.com/andre/ssl-cert-test/internal/discovery"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchDomainsFromCloudflare(t *testing.T) {
	// 1. Setup Mock Cloudflare Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the URL path matches what the library requests
		if r.URL.Path != "/zones/test-zone-id/dns_records" {
			http.NotFound(w, r)
			return
		}

		// Verify Authentication Header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// FIX: Verify Query Params contain the filter
		if r.URL.Query().Get("type") != "A,CNAME" {
			t.Errorf("Expected query param type=A,CNAME, got %s", r.URL.Query().Get("type"))
		}

		// Return Mock Response
		// FIX: Don't return MX records here. Real API wouldn't return them if type=A,CNAME is requested.
		response := map[string]interface{}{
			"success": true,
			"result": []map[string]interface{}{
				{
					"name": "example.com",
					"type": "A",
				},
				{
					"name": "www.example.com",
					"type": "CNAME",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	// 2. Monkey Patch the API URL
	originalURL := CloudflareBaseURL
	CloudflareBaseURL = ts.URL
	defer func() {
		CloudflareBaseURL = originalURL
	}()

	// 3. Run Function
	domains, err := FetchDomainsFromCloudflare("test-token", "test-zone-id")

	// 4. Assertions
	assert.NoError(t, err)
	assert.Len(t, domains, 2) // Should match the 2 items in mock result
	assert.Contains(t, domains, "example.com")
	assert.Contains(t, domains, "www.example.com")
}

func TestFetchDomainsFromCloudflare_Error(t *testing.T) {
	// Test server error
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	// Swap URL
	originalURL := CloudflareBaseURL
	CloudflareBaseURL = ts.URL
	defer func() { CloudflareBaseURL = originalURL }()

	_, err := FetchDomainsFromCloudflare("token", "zone")
	assert.Error(t, err)
}

func TestFetchDomainsFromCloudflare_MissingArgs(t *testing.T) {
	// Should fail fast without calling API
	_, err := FetchDomainsFromCloudflare("", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}
