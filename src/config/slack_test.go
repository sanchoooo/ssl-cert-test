package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendSlackAlert(t *testing.T) {
	// 1. Setup a Mock Slack Server
	// This captures the request your code sends so we can inspect it
	var receivedPayload SlackMessage
	var requestCount int

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Verify Request Headers
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode Payload to verify contents
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
			Domain:          "expire.com",
			Port:            443,
			CommonName:      "expire.com",
			IPAddress:       "1.2.3.4",
			DaysUntilExpiry: 3, // Critical: Below alertDays (5)
			NotAfter:        now.Add(72 * time.Hour),
		},
		{
			Domain:          "safe.com",
			Port:            443,
			DaysUntilExpiry: 30, // Safe: Should be ignored
		},
		{
			Domain:     "error.com",
			Port:       8443,
			Error:      "connection refused", // Error: Should be included
			CommonName: "error.com",
		},
	}

	// 3. Run Function using the Mock URL
	// We pass ts.URL instead of a real slack.com URL
	err := SendSlackAlert(ts.URL, 5, testData)

	// 4. Assertions
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount, "Should have sent exactly 1 HTTP request")
	
	// Verify the summary text counts correctly (1 expiring + 1 error = 2 issues)
	assert.Contains(t, receivedPayload.Text, "Found 2 SSL Certificates", "Summary text should match issue count")
	assert.Len(t, receivedPayload.Attachments, 2, "Should have 2 attachments (expire.com and error.com)")

	// Detailed check on the first attachment (expire.com)
	att1 := receivedPayload.Attachments[0]
	assert.Equal(t, "warning", att1.Color)
	assert.Contains(t, att1.Title, "expire.com")
	assert.Contains(t, att1.Text, "Expiring in 3 days")

	// Detailed check on the second attachment (error.com)
	att2 := receivedPayload.Attachments[1]
	assert.Equal(t, "danger", att2.Color)
	assert.Contains(t, att2.Title, "error.com")
	assert.Contains(t, att2.Text, "Error: connection refused")
}

func TestSendSlackAlert_NoExpiry(t *testing.T) {
	// Setup Mock Server that should FAIL if called
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Slack webhook should NOT be called when no domains are expiring")
	}))
	defer ts.Close()

	testData := []DomainValidity{
		{Domain: "safe.com", DaysUntilExpiry: 30},
	}

	// Run with alertDays=5. Since 30 > 5, nothing should send.
	err := SendSlackAlert(ts.URL, 5, testData)
	assert.NoError(t, err)
}

func TestSendSlackAlert_NoWebhook(t *testing.T) {
	// If webhook is empty, it should simply return nil without doing anything
	err := SendSlackAlert("", 5, nil)
	assert.NoError(t, err)
}