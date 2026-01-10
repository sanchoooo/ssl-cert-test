package discovery

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/stretchr/testify/assert"
)

// MockAzurePager implements AzurePagingClient
type MockAzurePager struct {
	pages     []armdns.RecordSetsClientListByDNSZoneResponse
	pageIndex int
}

func (m *MockAzurePager) More() bool {
	return m.pageIndex < len(m.pages)
}

func (m *MockAzurePager) NextPage(ctx context.Context) (armdns.RecordSetsClientListByDNSZoneResponse, error) {
	p := m.pages[m.pageIndex]
	m.pageIndex++
	return p, nil
}

func TestFetchDomainsFromAzure(t *testing.T) {
	// 1. Setup Mock Data
	zoneName := "azure-test.com"

	strPtr := func(s string) *string { return &s }

	mockResponse := armdns.RecordSetsClientListByDNSZoneResponse{
		RecordSetListResult: armdns.RecordSetListResult{
			Value: []*armdns.RecordSet{
				{
					Name: strPtr("www"),
					Properties: &armdns.RecordSetProperties{
						// FIX: Changed Ipv4Address to IPv4Address
						ARecords: []*armdns.ARecord{{IPv4Address: strPtr("1.2.3.4")}},
					},
				},
				{
					Name: strPtr("@"),
					Properties: &armdns.RecordSetProperties{
						// FIX: Changed Ipv4Address to IPv4Address
						ARecords: []*armdns.ARecord{{IPv4Address: strPtr("5.6.7.8")}},
					},
				},
				{
					Name: strPtr("api"),
					Properties: &armdns.RecordSetProperties{
						CnameRecord: &armdns.CnameRecord{Cname: strPtr("lb.azure.com")},
					},
				},
				{
					Name: strPtr("mail"),
					Properties: &armdns.RecordSetProperties{
						MxRecords: []*armdns.MxRecord{{Exchange: strPtr("mail.server")}},
					},
				},
			},
		},
	}

	// 2. Monkey Patch
	originalCreator := AzureClientCreatorFunc
	AzureClientCreatorFunc = func(subID, rg, zone, cID, cSecret, tID string) (AzurePagingClient, error) {
		return &MockAzurePager{
			pages: []armdns.RecordSetsClientListByDNSZoneResponse{mockResponse},
		}, nil
	}
	defer func() { AzureClientCreatorFunc = originalCreator }()

	// 3. Run Function
	domains, err := FetchDomainsFromAzure("sub", "rg", zoneName, "client", "secret", "tenant")

	// 4. Assertions
	assert.NoError(t, err)
	assert.Len(t, domains, 3)
	assert.Contains(t, domains, "www.azure-test.com")
	assert.Contains(t, domains, "azure-test.com")
	assert.Contains(t, domains, "api.azure-test.com")
}

func TestFetchDomainsFromAzure_MissingArgs(t *testing.T) {
	_, err := FetchDomainsFromAzure("", "", "", "", "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}
