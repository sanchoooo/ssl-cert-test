package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// TeamsMessage represents the legacy MessageCard format for Teams Webhooks
type TeamsMessage struct {
	Type       string         `json:"@type"`
	Context    string         `json:"@context"`
	ThemeColor string         `json:"themeColor"`
	Summary    string         `json:"summary"`
	Sections   []TeamsSection `json:"sections"`
}

type TeamsSection struct {
	ActivityTitle    string      `json:"activityTitle"`
	ActivitySubtitle string      `json:"activitySubtitle"`
	Facts            []TeamsFact `json:"facts"`
	Markdown         bool        `json:"markdown"`
}

type TeamsFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SendTeamsAlert sends a formatted card to MS Teams
func SendTeamsAlert(webhookURL string, alertDays int, data []DomainValidity) error {
	if webhookURL == "" {
		return nil
	}

	var expiring []DomainValidity
	for _, r := range data {
		if r.Error != "" || r.DaysUntilExpiry <= alertDays {
			expiring = append(expiring, r)
		}
	}

	if len(expiring) == 0 {
		return nil
	}

	// Build the Card
	summary := fmt.Sprintf("⚠️ Found %d SSL Certificates expiring within %d days", len(expiring), alertDays)
	
	var sections []TeamsSection
	for _, r := range expiring {
		title := fmt.Sprintf("%s (Port: %d)", r.Domain, r.Port)
		
		status := fmt.Sprintf("Expiring in %d days", r.DaysUntilExpiry)
		if r.Error != "" {
			status = fmt.Sprintf("Error: %s", r.Error)
		}

		sections = append(sections, TeamsSection{
			ActivityTitle:    title,
			ActivitySubtitle: status,
			Markdown:         true,
			Facts: []TeamsFact{
				{Name: "Common Name", Value: r.CommonName},
				{Name: "IP Address", Value: r.IPAddress},
				{Name: "Not After", Value: r.NotAfter.Format("2006-01-02")},
				{Name: "Chain Status", Value: r.ChainStatus},
			},
		})
	}

	payload := TeamsMessage{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: "d70000", // Red color for alerts
		Summary:    summary,
		Sections:   sections,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal teams payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send teams request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("teams API returned non-200 status: %s", resp.Status)
	}

	fmt.Println("Teams alert sent successfully.")
	return nil
}