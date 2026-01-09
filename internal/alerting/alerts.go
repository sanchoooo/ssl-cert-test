package alerting

import (
	"fmt"
)

// AlertProvider is the common interface for all notification channels
type AlertProvider interface {
	Send(results []DomainValidity, alertDays int) error
	Name() string
}

// -- Implementations --

// 1. PagerDuty
type PagerDutyAlert struct {
	IntegrationKey string
}

func (p *PagerDutyAlert) Name() string { return "PagerDuty" }

func (p *PagerDutyAlert) Send(results []DomainValidity, alertDays int) error {
	if p.IntegrationKey == "" {
		return nil
	}
	return SendAlert(p.IntegrationKey, alertDays, results)
}

// 2. Slack
type SlackAlert struct {
	WebhookURL string
}

func (s *SlackAlert) Name() string { return "Slack" }

func (s *SlackAlert) Send(results []DomainValidity, alertDays int) error {
	if s.WebhookURL == "" {
		return nil
	}
	return SendSlackAlert(s.WebhookURL, alertDays, results)
}

// 3. Teams
type TeamsAlert struct {
	WebhookURL string
}

func (t *TeamsAlert) Name() string { return "Teams" }

func (t *TeamsAlert) Send(results []DomainValidity, alertDays int) error {
	if t.WebhookURL == "" {
		return nil
	}
	return SendTeamsAlert(t.WebhookURL, alertDays, results)
}

// -- Factory --

// GetAlertProviders returns a list of configured alert providers
func GetAlertProviders(cfg *AppConfig) []AlertProvider {
	var providers []AlertProvider

	if cfg.PagerDutyKey != "" {
		providers = append(providers, &PagerDutyAlert{IntegrationKey: cfg.PagerDutyKey})
	}
	if cfg.SlackWebhook != "" {
		providers = append(providers, &SlackAlert{WebhookURL: cfg.SlackWebhook})
	}
	if cfg.TeamsWebhook != "" {
		providers = append(providers, &TeamsAlert{WebhookURL: cfg.TeamsWebhook})
	}

	return providers
}
