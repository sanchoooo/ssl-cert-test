package scan

import (
	"context" // <--- Added
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSSLValidity_TLSVersions(t *testing.T) {
	tests := []struct {
		name        string
		minVersion  uint16
		maxVersion  uint16
		wantVersion string
	}{
		{
			name:        "Detect TLS 1.2",
			minVersion:  tls.VersionTLS12,
			maxVersion:  tls.VersionTLS12,
			wantVersion: "TLS 1.2",
		},
		{
			name:        "Detect TLS 1.3",
			minVersion:  tls.VersionTLS13,
			maxVersion:  tls.VersionTLS13,
			wantVersion: "TLS 1.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			ts.TLS = &tls.Config{
				MinVersion: tt.minVersion,
				MaxVersion: tt.maxVersion,
			}
			ts.StartTLS()
			defer ts.Close()

			u, _ := url.Parse(ts.URL)
			host := u.Hostname()
			port, _ := strconv.Atoi(u.Port())

			// Updated call with Context
			details, err := GetSSLValidity(context.Background(), host, port)

			assert.NoError(t, err)
			assert.Equal(t, tt.wantVersion, details.TLSVersion)
		})
	}
}
