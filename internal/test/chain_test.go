package test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Helper to generate a cert template
func createCertTemplate(isCA bool, subject string, issuer *x509.Certificate) (*x509.Certificate, *rsa.PrivateKey) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	notBefore := time.Now()
	notAfter := notBefore.Add(1 * time.Hour)

	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: subject,
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  isCA,
		DNSNames:              []string{"example.com", "127.0.0.1"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	return template, priv
}

func TestGetSSLValidity_ChainValidation(t *testing.T) {
	// 1. Setup PKI: Define these variables at the top level of the test

	// A. Create Root CA
	rootTmpl, rootKey := createCertTemplate(true, "Root CA", nil)
	_, _ = x509.CreateCertificate(rand.Reader, rootTmpl, rootTmpl, &rootKey.PublicKey, rootKey)

	// B. Create Intermediate CA (Signed by Root)
	intTmpl, intKey := createCertTemplate(true, "Intermediate CA", rootTmpl)
	intDER, _ := x509.CreateCertificate(rand.Reader, intTmpl, rootTmpl, &intKey.PublicKey, rootKey)

	// C. Create Leaf Cert (Signed by Intermediate)
	leafTmpl, leafKey := createCertTemplate(false, "Leaf Cert", intTmpl)
	leafDER, _ := x509.CreateCertificate(rand.Reader, leafTmpl, intTmpl, &leafKey.PublicKey, intKey)

	// 2. Configure Test Cases
	tests := []struct {
		name         string
		sendChain    bool
		expectStatus string
	}{
		{
			name:         "Broken Chain (Missing Intermediate)",
			sendChain:    false,
			expectStatus: "Untrusted Root / Missing Intermediate",
		},
		{
			name:         "Valid Chain (Server sends Intermediate)",
			sendChain:    true,
			expectStatus: "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverCert := tls.Certificate{
				Certificate: [][]byte{leafDER}, // accessing outer variable
				PrivateKey:  leafKey,           // accessing outer variable
			}
			if tt.sendChain {
				serverCert.Certificate = append(serverCert.Certificate, intDER) // accessing outer variable
			}

			ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
			ts.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
			ts.StartTLS()
			defer ts.Close()

			u, _ := url.Parse(ts.URL)
			host := u.Hostname()
			port, _ := strconv.Atoi(u.Port())

			// Updated to pass Context
			details, err := getSSLValidity(context.Background(), host, port)

			assert.NoError(t, err)

			if tt.sendChain {
				assert.Contains(t, details.ChainStatus, "Untrusted Root")
			} else {
				assert.Contains(t, details.ChainStatus, "Untrusted Root")
			}
		})
	}
}
