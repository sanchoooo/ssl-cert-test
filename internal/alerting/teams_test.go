package test

import (
	"encoding/json"
	"github.com/andre/ssl-cert-test/internal/config"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendTeamsAlert(t *testing.T) {
	// 1. Setup Mock Teams Server
	var receivedPayload TeamsMessage
	var requestCount int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Teams webhooks expect POST
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Decode Payload
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// 2. Define Test Data
	now := time.Now()
	testData := []DomainValidity{
		{
			Domain:          "expire.teams.com",
			Port:            443,
			CommonName:      "expire.teams.com",
			IPAddress:       "10.0.0.1",
			DaysUntilExpiry: 3, // Critical
			ChainStatus:     "OK",
			NotAfter:        now.Add(72 * time.Hour),
		},
		{
			Domain:          "safe.teams.com",
			Port:            443,
			DaysUntilExpiry: 60, // Safe
		},
		{
			Domain:      "broken-chain.com",
			Port:        443,
			CommonName:  "broken-chain.com",
			ChainStatus: "Missing Intermediate",
			Error:       "Untrusted Root",
			IPAddress:   "10.0.0.2",
		},
	}

	// 3. Run Function
	err := SendTeamsAlert(ts.URL, 5, testData)

	// 4. Assertions
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount)

	// Check Top Level Fields
	assert.Equal(t, "MessageCard", receivedPayload.Type)
	assert.Equal(t, "d70000", receivedPayload.ThemeColor)
	assert.Contains(t, receivedPayload.Summary, "Found 2 SSL Certificates")

	// Check Sections (Should have 2: expire.teams.com and broken-chain.com)
	assert.Len(t, receivedPayload.Sections, 2)

	// Verify First Section (Expiry)
	sec1 := receivedPayload.Sections[0]
	assert.Contains(t, sec1.ActivityTitle, "expire.teams.com")
	assert.Contains(t, sec1.ActivitySubtitle, "Expiring in 3 days")

	// Verify Facts in Section 1
	assert.Equal(t, "Common Name", sec1.Facts[0].Name)
	assert.Equal(t, "expire.teams.com", sec1.Facts[0].Value)
	assert.Equal(t, "Chain Status", sec1.Facts[3].Name)
	assert.Equal(t, "OK", sec1.Facts[3].Value)

	// Verify Second Section (Broken Chain Error)
	sec2 := receivedPayload.Sections[1]
	assert.Contains(t, sec2.ActivityTitle, "broken-chain.com")
	assert.Contains(t, sec2.ActivitySubtitle, "Error: Untrusted Root")
}

func TestSendTeamsAlert_NoWebhook(t *testing.T) {
	err := SendTeamsAlert("", 5, nil)
	assert.NoError(t, err)
}
