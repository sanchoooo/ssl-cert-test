package discovery

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

func FetchDomainsFromRoute53(hostedZoneID string) ([]string, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %v", err)
	}

	svc := route53.New(sess)
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
	}
	var domains []string

	err = svc.ListResourceRecordSetsPages(input, func(rrPage *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
		for _, record := range rrPage.ResourceRecordSets {
			if aws.StringValue(record.Type) == "A" || aws.StringValue(record.Type) == "CNAME" {
				domains = append(domains, aws.StringValue(record.Name))
			}
		}
		return !lastPage
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list resource record sets: %v", err)
	}

	return domains, nil
}
