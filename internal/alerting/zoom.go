package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andre/ssl-cert-test/internal/config"
)

type ZoomAlert struct {
	WebhookURL string
}

func (z *ZoomAlert) Name() string { return "Zoom" }

// Zoom Payload Structures for "format=full"
type zoomPayload struct {
	Content zoomContent `json:"content"`
}

type zoomContent struct {
	Head zoomHead   `json:"head"`
	Body []zoomBody `json:"body"`
}

type zoomHead struct {
	Text string `json:"text"`
}

type zoomBody struct {
	Type string `json:"type"` // "message"
	Text string `json:"text"`
}

func (z *ZoomAlert) Send(results []config.DomainValidity, alertDays int) error {
	var expired []config.DomainValidity
	for _, r := range results {
		if r.DaysUntilExpiry <= alertDays {
			expired = append(expired, r)
		}
	}

	if len(expired) == 0 {
		return nil
	}

	// Build the message text
	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("The following certificates expire within %d days:\n", alertDays))
	for _, e := range expired {
		msgBuilder.WriteString(fmt.Sprintf("- %s (Expires in %d days)\n", e.Domain, e.DaysUntilExpiry))
	}

	// Construct payload
	payload := zoomPayload{
		Content: zoomContent{
			Head: zoomHead{
				Text: "SSL Certificate Alert",
			},
			Body: []zoomBody{
				{
					Type: "message",
					Text: msgBuilder.String(),
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal zoom payload: %w", err)
	}

	// Ensure the URL has format=full for structured JSON
	url := z.WebhookURL
	if !strings.Contains(url, "format=") {
		if strings.Contains(url, "?") {
			url += "&format=full"
		} else {
			url += "?format=full"
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Some Zoom webhooks require an Authorization header if not embedded in the URL,
	// but the standard "Incoming Webhook" app includes the token in the URL path.

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send zoom alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("received bad status from zoom: %s", resp.Status)
	}

	return nil
}
