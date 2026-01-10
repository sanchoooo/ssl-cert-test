package alerting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andre/ssl-cert-test/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestZoomAlert_Send(t *testing.T) {
	// 1. Start Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL param
		if !strings.Contains(r.URL.RawQuery, "format=full") {
			t.Errorf("Expected URL to contain format=full, got: %s", r.URL.String())
		}

		// Verify Payload
		var payload zoomPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode payload: %v", err)
		}

		assert.Equal(t, "SSL Certificate Alert", payload.Content.Head.Text)
		assert.NotEmpty(t, payload.Content.Body)
		assert.Contains(t, payload.Content.Body[0].Text, "example.com")

		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// 2. Configure Alert
	alert := ZoomAlert{WebhookURL: ts.URL} // URL without params

	// 3. Create Dummy Results
	results := []config.DomainValidity{
		{
			Domain:          "example.com",
			DaysUntilExpiry: 2,
			NotAfter:        time.Now().Add(48 * time.Hour),
		},
		{
			Domain:          "safe.com",
			DaysUntilExpiry: 30,
			NotAfter:        time.Now().Add(720 * time.Hour),
		},
	}

	// 4. Send Alert
	err := alert.Send(results, 5)
	assert.NoError(t, err)
}

func TestZoomAlert_NoExpiry(t *testing.T) {
	alert := ZoomAlert{WebhookURL: "http://unused"}
	results := []config.DomainValidity{
		{Domain: "safe.com", DaysUntilExpiry: 20},
	}
	err := alert.Send(results, 5)
	assert.NoError(t, err)
}
