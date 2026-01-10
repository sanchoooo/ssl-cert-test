package discovery

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime" // <--- Added Import
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
)

// AzurePagingClient interface allows us to mock the paginator in tests
type AzurePagingClient interface {
	More() bool
	NextPage(ctx context.Context) (armdns.RecordSetsClientListByDNSZoneResponse, error)
}

// RealAzurePager wraps the SDK's generic pager to satisfy our interface
type RealAzurePager struct {
	// FIX: Use the generic runtime.Pager instead of the undefined named type
	pager *runtime.Pager[armdns.RecordSetsClientListByDNSZoneResponse]
}

func (p *RealAzurePager) More() bool {
	return p.pager.More()
}

func (p *RealAzurePager) NextPage(ctx context.Context) (armdns.RecordSetsClientListByDNSZoneResponse, error) {
	return p.pager.NextPage(ctx)
}

// AzureClientCreatorFunc is a variable we can swap in tests to return a mock client
var AzureClientCreatorFunc = func(subID, rg, zone, cID, cSecret, tID string) (AzurePagingClient, error) {
	cred, err := azidentity.NewClientSecretCredential(tID, cID, cSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid azure credentials: %w", err)
	}

	client, err := armdns.NewRecordSetsClient(subID, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azure dns client: %w", err)
	}

	// Create the Pager provided by the SDK (Using DNSZone capitalized)
	pager := client.NewListByDNSZonePager(rg, zone, nil)
	return &RealAzurePager{pager: pager}, nil
}

// FetchDomainsFromAzure retrieves A and CNAME records from an Azure DNS Zone
func FetchDomainsFromAzure(subID, rg, zone, cID, cSecret, tID string) ([]string, error) {
	if subID == "" || rg == "" || zone == "" || cID == "" || cSecret == "" || tID == "" {
		return nil, fmt.Errorf("missing required azure configuration fields")
	}

	// Use the creator function (which might be real or mocked)
	pager, err := AzureClientCreatorFunc(subID, rg, zone, cID, cSecret, tID)
	if err != nil {
		return nil, err
	}

	var domains []string
	ctx := context.Background()

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list dns records: %w", err)
		}

		for _, record := range page.Value {
			// In Azure SDK, record.Name is relative (e.g., "www"), not FQDN
			// We construct the FQDN manually.

			// Simple check for A or CNAME presence in the struct properties
			isA := record.Properties.ARecords != nil && len(record.Properties.ARecords) > 0
			isCNAME := record.Properties.CnameRecord != nil

			if isA || isCNAME {
				name := *record.Name
				if name == "@" {
					domains = append(domains, zone)
				} else {
					domains = append(domains, fmt.Sprintf("%s.%s", name, zone))
				}
			}
		}
	}

	return domains, nil
}
