package scan

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/andre/ssl-cert-test/internal/config"
)

// CertDetails holds the raw certificate data
type CertDetails struct {
	NotBefore     time.Time
	NotAfter      time.Time
	CommonName    string
	Serial        string
	TLSVersion    string
	ChainStatus   string
	CipherSuite   string
	FIPSCompliant bool
	Issuer        string
	SignatureAlgo string
	SANs          []string
}

func checkFIPSCompliance(version uint16, cipher uint16) bool {
	if version != tls.VersionTLS12 && version != tls.VersionTLS13 {
		return false
	}
	switch cipher {
	case tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:
		return true
	}
	return false
}

func tlsVersionToString(ver uint16) string {
	switch ver {
	case tls.VersionTLS13:
		return "TLS 1.3"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionSSL30:
		return "SSL 3.0"
	default:
		return fmt.Sprintf("Unknown (%x)", ver)
	}
}

// GetSSLValidity now takes a Context for timeout/cancellation
func GetSSLValidity(ctx context.Context, domain string, port int) (CertDetails, error) {
	var details CertDetails
	address := fmt.Sprintf("%s:%d", domain, port)

	// Use Dialer with Context
	dialer := &net.Dialer{}

	// 1. Establish TCP connection with Context
	rawConn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return details, fmt.Errorf("failed to connect: %v", err)
	}
	defer rawConn.Close()

	// 2. Upgrade to TLS
	conn := tls.Client(rawConn, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         domain, // SNI support
	})

	// 3. Handshake with Context (Go 1.17+)
	if err := conn.HandshakeContext(ctx); err != nil {
		return details, fmt.Errorf("handshake failed: %v", err)
	}

	// 4. Extract Data
	state := conn.ConnectionState()
	details.TLSVersion = tlsVersionToString(state.Version)
	details.CipherSuite = tls.CipherSuiteName(state.CipherSuite)
	details.FIPSCompliant = checkFIPSCompliance(state.Version, state.CipherSuite)

	certs := state.PeerCertificates
	if len(certs) == 0 {
		return details, errors.New("no certificates found")
	}

	leaf := certs[0]
	details.NotBefore = leaf.NotBefore
	details.NotAfter = leaf.NotAfter
	details.CommonName = leaf.Subject.CommonName
	details.Serial = leaf.SerialNumber.Text(16)
	details.SANs = leaf.DNSNames
	details.SignatureAlgo = leaf.SignatureAlgorithm.String()

	details.Issuer = leaf.Issuer.CommonName
	if details.Issuer == "" {
		if len(leaf.Issuer.Organization) > 0 {
			details.Issuer = leaf.Issuer.Organization[0]
		} else {
			details.Issuer = "Unknown"
		}
	}

	// 5. Chain Validation
	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}

	opts := x509.VerifyOptions{
		DNSName:       domain,
		Intermediates: intermediates,
	}

	details.ChainStatus = "OK"
	if _, err := leaf.Verify(opts); err != nil {
		switch e := err.(type) {
		case x509.UnknownAuthorityError:
			details.ChainStatus = "Untrusted Root / Missing Intermediate"
		case x509.HostnameError:
			details.ChainStatus = fmt.Sprintf("Hostname Mismatch: %s", e.Error())
		case x509.CertificateInvalidError:
			details.ChainStatus = fmt.Sprintf("Invalid Cert: %s", e.Error())
		default:
			details.ChainStatus = fmt.Sprintf("Chain Error: %v", err)
		}
	}

	return details, nil
}

// ProcessDomains now accepts a parent Context and uses slog
func ProcessDomains(ctx context.Context, domains []string, ports []int, timeout time.Duration, now time.Time, resultsChan chan<- config.DomainValidity, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create a child logger for this batch if needed, or use default
	logger := slog.Default()

	for _, domain := range domains {
		for _, port := range ports {
			logger.Debug("scanning target", "domain", domain, "port", port)

			// Create a per-request context with timeout
			reqCtx, cancel := context.WithTimeout(ctx, timeout)

			// Call updated function
			details, err := GetSSLValidity(reqCtx, domain, port)
			cancel() // Clean up context immediately

			result := config.DomainValidity{
				Domain:        domain,
				Port:          port,
				Serial:        details.Serial,
				TLSVersion:    details.TLSVersion,
				CipherSuite:   details.CipherSuite,
				FIPSCompliant: details.FIPSCompliant,
				ChainStatus:   details.ChainStatus,
				Issuer:        details.Issuer,
				SignatureAlgo: details.SignatureAlgo,
				SANs:          details.SANs,
				NotBefore:     details.NotBefore,
				NotAfter:      details.NotAfter,
				CommonName:    details.CommonName,
			}

			if err == nil {
				result.DaysUntilExpiry = int(details.NotAfter.Sub(now).Hours() / 24)
			} else {
				result.Error = err.Error()
				result.DaysUntilExpiry = 999999
				logger.Warn("scan failed", "domain", domain, "error", err)
			}
			resultsChan <- result
		}
	}
}
