package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// https://developer.pagerduty.com/docs/send-alert-event
var (
	pagerDutyEventsAPI = "https://events.pagerduty.com/v2/enqueue"
)

// PagerDutyEvent represents the structure of an Events API v2 payload
type PagerDutyEvent struct {
	RoutingKey  string                `json:"routing_key"`
	EventAction string                `json:"event_action"`
	DedupKey    string                `json:"dedup_key,omitempty"`
	Payload     PagerDutyEventPayload `json:"payload"`
	Client      string                `json:"client,omitempty"`
	ClientURL   string                `json:"client_url,omitempty"`
	Links       []PagerDutyEventLink  `json:"links,omitempty"`
	Images      []PagerDutyEventImage `json:"images,omitempty"`
}

// PagerDutyEventPayload represents the payload details of a PagerDuty event
type PagerDutyEventPayload struct {
	Summary       string                 `json:"summary"`
	Source        string                 `json:"source"`
	Severity      string                 `json:"severity"`
	Timestamp     string                 `json:"timestamp,omitempty"`
	Component     string                 `json:"component,omitempty"`
	Group         string                 `json:"group,omitempty"`
	Class         string                 `json:"class,omitempty"`
	CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}

// PagerDutyEventLink represents a link associated with the event
type PagerDutyEventLink struct {
	Href string `json:"href"`
	Text string `json:"text,omitempty"`
}

// PagerDutyEventImage represents an image associated with the event
type PagerDutyEventImage struct {
	Src  string `json:"src"`
	Href string `json:"href,omitempty"`
	Alt  string `json:"alt,omitempty"`
}

func SendAlert(integrationKey string, alertDays int, data []DomainValidity) error {

	if integrationKey == "" {
		log.Println("pagerdutykey environment variable not set")
	}

	for _, r := range data {

		if alertDays < r.DaysUntilExpiry {
			continue
		}
		eventPayload := PagerDutyEventPayload{
			Summary:   fmt.Sprintf("Certificate Expiration - %s using %s", r.Domain, r.CommonName),
			Source:    "cert-check",
			Severity:  "info",
			Component: "Certificate",
			CustomDetails: map[string]interface{}{
				"Domain":          r.Domain,
				"Port":            fmt.Sprint(r.Port),
				"NotAfter":        fmt.Sprint(r.NotAfter),
				"DaysUntilExpiry": fmt.Sprint(r.DaysUntilExpiry),
				"CommonName":      r.CommonName,
			},
		}

		// Create the PagerDuty event
		event := PagerDutyEvent{
			RoutingKey:  integrationKey,
			EventAction: "trigger",    // Can be "trigger", "acknowledge", or "resolve"
			DedupKey:    r.CommonName, // Optional: used for de-duplication
			Payload:     eventPayload,
			Client:      "cert-check",
		}

		// Marshal the event struct to JSON
		jsonPayload, err := json.Marshal(event)
		if err != nil {
			fmt.Printf("Error marshaling JSON: %v\n", err)
			return nil
		}

		// Create an HTTP POST request
		req, err := http.NewRequest("POST", pagerDutyEventsAPI, bytes.NewBuffer(jsonPayload))
		if err != nil {
			fmt.Printf("Error creating request: %v\n", err)
			return nil
		}

		// Set the Content-Type header
		req.Header.Set("Content-Type", "application/json")

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error sending request: %v\n", err)
		}
		defer resp.Body.Close()

		// Check the response status
		if resp.StatusCode != http.StatusAccepted {
			fmt.Printf("PagerDuty API returned an error: %s\n", resp.Status)
			return nil
		}

		fmt.Println("PagerDuty event successfully sent for ", event)
	}
	return nil

}
