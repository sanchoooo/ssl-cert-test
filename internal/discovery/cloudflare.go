package discovery

import (
	"context"
	"fmt"
	"github.com/andre/ssl-cert-test/internal/config"
	"github.com/cloudflare/cloudflare-go"
)

// Exported variable to allow overriding in tests
var CloudflareBaseURL = "https://api.cloudflare.com/client/v4"

// FetchDomainsFromCloudflare retrieves A and CNAME records from a Cloudflare Zone
func FetchDomainsFromCloudflare(apiToken, zoneID string) ([]string, error) {
	if apiToken == "" || zoneID == "" {
		return nil, fmt.Errorf("cloudflare token and zone ID are required")
	}

	// FIX: Use NewWithAPIToken instead of New
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloudflare client: %w", err)
	}

	// Override BaseURL if needed (for testing)
	if CloudflareBaseURL != "https://api.cloudflare.com/client/v4" {
		api.BaseURL = CloudflareBaseURL
	}

	ctx := context.Background()

	// List DNS Records
	records, _, err := api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
		Type: "A,CNAME",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch DNS records: %w", err)
	}

	var domains []string
	for _, r := range records {
		if r.Name != "" {
			domains = append(domains, r.Name)
		}
	}

	return domains, nil
}
