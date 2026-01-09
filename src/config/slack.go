package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// SlackMessage represents the payload sent to Slack
type SlackMessage struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color  string `json:"color"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Footer string `json:"footer"`
}

// SendSlackAlert sends a summary of expiring certificates to Slack
func SendSlackAlert(webhookURL string, alertDays int, data []DomainValidity) error {
	if webhookURL == "" {
		return nil
	}

	var expiring []DomainValidity
	for _, r := range data {
		// If error exists or days until expiry is less than threshold
		if r.Error != "" || r.DaysUntilExpiry <= alertDays {
			expiring = append(expiring, r)
		}
	}

	// If nothing is expiring, don't send anything
	if len(expiring) == 0 {
		return nil
	}

	// Build the message
	messageText := fmt.Sprintf("⚠️ Found %d SSL Certificates expiring within %d days (or errors)", len(expiring), alertDays)
	
	var attachments []Attachment

	for _, r := range expiring {
		color := "warning"
		status := fmt.Sprintf("Expiring in %d days", r.DaysUntilExpiry)
		
		if r.Error != "" {
			color = "danger"
			status = fmt.Sprintf("Error: %s", r.Error)
		} else if r.DaysUntilExpiry < 2 {
			color = "danger"
		}

		attachments = append(attachments, Attachment{
			Color:  color,
			Title:  fmt.Sprintf("%s (Port: %d)", r.Domain, r.Port),
			Text:   fmt.Sprintf("Common Name: %s\nStatus: %s\nIP: %s", r.CommonName, status, r.IPAddress),
			Footer: "SSL Cert Checker",
		})
	}

	payload := SlackMessage{
		Text:        messageText,
		Attachments: attachments,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send slack request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned non-200 status: %s", resp.Status)
	}

	fmt.Println("Slack alert sent successfully.")
	return nil
}