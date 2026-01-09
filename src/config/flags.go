package config

import (
	"flag"
	"fmt"
	"time"

	"github.com/peterbourgon/ff/v3"
)

// AppConfig holds all startup configuration/flags
type AppConfig struct {
	// ... existing fields ...
	ConfigFile string
	Output     string
	Verbose    bool
	Help       bool

	// Tuning
	Timeout time.Duration
	Split   int

	// Logic Config
	ConfigType   string // "zone", "config", "gitlab", "cloudflare", "azure" <--- Added azure
	PortString   string
	HostedZoneID string

	// Cloudflare
	CloudflareToken  string
	CloudflareZoneID string

	// Azure Specifics
	AzureClientID       string // <--- NEW
	AzureClientSecret   string // <--- NEW
	AzureTenantID       string // <--- NEW
	AzureSubscriptionID string // <--- NEW
	AzureResourceGroup  string // <--- NEW
	AzureDNSZone        string // <--- NEW

	// Alerting
	PagerDutyKey string
	SlackWebhook string
	TeamsWebhook string
	AlertDays    int

	// Gitlab
	GitlabToken     string
	GitlabURL       string
	GitlabProjectID string
	GitlabFilePath  string
	GitlabRef       string
}

// Load parses flags and environment variables
func Load(args []string) (*AppConfig, error) {
	fs := flag.NewFlagSet("ssl_checker", flag.ExitOnError)

	var cfg AppConfig

	// ... existing flags ...
	fs.StringVar(&cfg.ConfigFile, "config", "", "Path to the configuration file")
	fs.StringVar(&cfg.Output, "outputfile", "/tmp/data", "Write output data to the specified file")
	fs.BoolVar(&cfg.Verbose, "verbose", true, "Enable verbose mode")
	fs.BoolVar(&cfg.Help, "help", false, "Display help message")

	fs.DurationVar(&cfg.Timeout, "timeout", 5*time.Second, "Timeout for connection attempts")
	fs.IntVar(&cfg.Split, "split", 30, "Number of domains to test per thread")

	fs.StringVar(&cfg.ConfigType, "type", "gitlab", "Which config to use: zone, config, gitlab, cloudflare, azure")
	fs.StringVar(&cfg.PortString, "ports", "", "Comma-separated list of ports")
	fs.StringVar(&cfg.HostedZoneID, "hosted-zone-id", "", "Route53 Hosted Zone ID")

	// Cloudflare
	fs.StringVar(&cfg.CloudflareToken, "cloudflaretoken", "", "Cloudflare API Token")
	fs.StringVar(&cfg.CloudflareZoneID, "cloudflarezoneid", "", "Cloudflare Zone ID")

	// Azure
	fs.StringVar(&cfg.AzureClientID, "azureclientid", "", "Azure Client ID")
	fs.StringVar(&cfg.AzureClientSecret, "azureclientsecret", "", "Azure Client Secret")
	fs.StringVar(&cfg.AzureTenantID, "azuretenantid", "", "Azure Tenant ID")
	fs.StringVar(&cfg.AzureSubscriptionID, "azuresubscriptionid", "", "Azure Subscription ID")
	fs.StringVar(&cfg.AzureResourceGroup, "azureresourcegroup", "", "Azure Resource Group Name")
	fs.StringVar(&cfg.AzureDNSZone, "azurezone", "", "Azure DNS Zone Name")

	// ... Alerting & Gitlab flags ...
	fs.StringVar(&cfg.PagerDutyKey, "pagerdutykey", "", "PagerDuty Integration Key")
	fs.StringVar(&cfg.SlackWebhook, "slackwebhook", "", "Slack Webhook URL")
	fs.StringVar(&cfg.TeamsWebhook, "teamswebhook", "", "MS Teams Webhook URL")
	fs.IntVar(&cfg.AlertDays, "alertdays", 5, "Number of days remaining to trigger alert")

	fs.StringVar(&cfg.GitlabToken, "gitlabtoken", "", "Gitlab Token")
	fs.StringVar(&cfg.GitlabURL, "gitlaburl", "", "Gitlab Instance URL")
	fs.StringVar(&cfg.GitlabProjectID, "gitlabprojectid", "", "Gitlab Project ID")
	fs.StringVar(&cfg.GitlabFilePath, "gitlabfilepath", "", "Path to YAML file in repo")
	fs.StringVar(&cfg.GitlabRef, "gitlabref", "", "Branch or commit SHA")

	err := ff.Parse(fs, args,
		ff.WithEnvVarPrefix("SSL"),
	)

	if err != nil {
		return nil, fmt.Errorf("error parsing configuration: %w", err)
	}

	return &cfg, nil
}
