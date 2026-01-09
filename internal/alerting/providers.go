package alerting

import (
	"fmt"
)

// TargetProvider is the common interface for all discovery methods
type TargetProvider interface {
	FetchTargets() (Config, error)
}

// -- Implementations --

// 1. FileSystem Provider
type FileProvider struct {
	Path string
}

func (p *FileProvider) FetchTargets() (Config, error) {
	return LoadConfig(p.Path)
}

// 2. Route53 Provider
type Route53Provider struct {
	HostedZoneID string
}

func (p *Route53Provider) FetchTargets() (Config, error) {
	domains, err := FetchDomainsFromRoute53(p.HostedZoneID)
	// Return a Config struct with just the domains populated
	return Config{Domains: domains}, err
}

// 3. Cloudflare Provider
type CloudflareProvider struct {
	Token  string
	ZoneID string
}

func (p *CloudflareProvider) FetchTargets() (Config, error) {
	domains, err := FetchDomainsFromCloudflare(p.Token, p.ZoneID)
	return Config{Domains: domains}, err
}

// 4. Azure Provider
type AzureProvider struct {
	SubID, ResGroup, Zone, ClientID, ClientSecret, TenantID string
}

func (p *AzureProvider) FetchTargets() (Config, error) {
	domains, err := FetchDomainsFromAzure(p.SubID, p.ResGroup, p.Zone, p.ClientID, p.ClientSecret, p.TenantID)
	return Config{Domains: domains}, err
}

// 5. GitLab Provider
type GitLabProvider struct {
	Token, URL, ProjectID, FilePath, Ref string
}

func (p *GitLabProvider) FetchTargets() (Config, error) {
	// We need to move the 'fetchGitLabConfig' logic we wrote in cert.go
	// into a reusable function in this package, or implement it here.
	// For now, assuming you move that helper logic to `gitlab.go` or similar:
	return FetchGitLabConfig(p.Token, p.URL, p.ProjectID, p.FilePath, p.Ref)
}

// -- Factory --

// GetProvider returns the correct provider based on configuration
func GetProvider(cfg *AppConfig) (TargetProvider, error) {
	switch cfg.ConfigType {
	case "zone":
		return &Route53Provider{HostedZoneID: cfg.HostedZoneID}, nil
	case "config":
		return &FileProvider{Path: cfg.ConfigFile}, nil
	case "cloudflare":
		return &CloudflareProvider{Token: cfg.CloudflareToken, ZoneID: cfg.CloudflareZoneID}, nil
	case "azure":
		return &AzureProvider{
			SubID:        cfg.AzureSubscriptionID,
			ResGroup:     cfg.AzureResourceGroup,
			Zone:         cfg.AzureDNSZone,
			ClientID:     cfg.AzureClientID,
			ClientSecret: cfg.AzureClientSecret,
			TenantID:     cfg.AzureTenantID,
		}, nil
	case "gitlab":
		return &GitLabProvider{
			Token:     cfg.GitlabToken,
			URL:       cfg.GitlabURL,
			ProjectID: cfg.GitlabProjectID,
			FilePath:  cfg.GitlabFilePath,
			Ref:       cfg.GitlabRef,
		}, nil
	default:
		return nil, fmt.Errorf("unknown config type: %s", cfg.ConfigType)
	}
}
