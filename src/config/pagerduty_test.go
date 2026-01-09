package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendAlert(t *testing.T) {
	// 1. Setup Mock PagerDuty Server
	// This captures the request your code sends
	var receivedEvent PagerDutyEvent
	requestCount := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Verify Method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Decode Payload
		if err := json.NewDecoder(r.Body).Decode(&receivedEvent); err != nil {
			t.Errorf("Failed to decode JSON payload: %v", err)
		}

		// PagerDuty expects 202 Accepted for successful events
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()

	// 2. Override the API URL (Monkey Patching)
	// We swap the real PagerDuty URL with our test server URL
	originalURL := pagerDutyEventsAPI
	pagerDutyEventsAPI = ts.URL
	defer func() {
		// Restore the original URL after test finishes
		pagerDutyEventsAPI = originalURL
	}()

	// 3. Define Test Data
	now := time.Now()
	testData := []DomainValidity{
		{
			Domain:          "critical.com",
			CommonName:      "critical.com",
			Port:            443,
			DaysUntilExpiry: 2, // Should TRIGGER (2 < 5)
			NotAfter:        now.Add(48 * time.Hour),
		},
		{
			Domain:          "safe.com",
			CommonName:      "safe.com",
			DaysUntilExpiry: 30, // Should SKIP (30 > 5)
		},
	}

	// 4. Run the function
	// We pass a fake integration key and an alert threshold of 5 days
	err := SendAlert("fake-integration-key", 5, testData)

	// 5. Assertions
	assert.NoError(t, err)
	assert.Equal(t, 1, requestCount, "Should have sent exactly 1 alert (for critical.com)")
	
	// Verify Payload Content
	assert.Equal(t, "fake-integration-key", receivedEvent.RoutingKey)
	assert.Equal(t, "trigger", receivedEvent.EventAction)
	assert.Equal(t, "cert-check", receivedEvent.Client)
	assert.Contains(t, receivedEvent.Payload.Summary, "critical.com", "Summary should contain the expiring domain")
	assert.Equal(t, "critical.com", receivedEvent.DedupKey, "DedupKey should match the CommonName")
}

func TestSendAlert_NoKey(t *testing.T) {
	// If no key is provided, it should log a warning but not crash/error
	// (Based on your current implementation logic)
	err := SendAlert("", 5, nil)
	assert.NoError(t, err)
}