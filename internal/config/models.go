package config

import (
	"time"
)

// Config holds the actual domains/ports to check
type Config struct {
	Ports   []int    `json:"ports"`
	Domains []string `json:"domains"`
	Cidr    []string `json:"cidr"`
}

// DomainValidity holds the scan results
type DomainValidity struct {
	Domain        string `json:"domain"`
	IPAddress     string `json:"ip_address"`
	Port          int    `json:"port"`
	Serial        string `json:"serial"`
	TLSVersion    string `json:"tls_version"`
	CipherSuite   string `json:"cipher_suite"`
	FIPSCompliant bool   `json:"fips_compliant"`
	ChainStatus   string `json:"chain_status"`

	// --- NEW FIELDS ---
	Issuer        string   `json:"issuer"`         // Who signed it?
	SignatureAlgo string   `json:"signature_algo"` // e.g., SHA256-RSA
	SANs          []string `json:"sans"`           // List of all valid domains
	// ------------------

	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	DaysUntilExpiry int       `json:"days_until_expiry"`
	CommonName      string    `json:"common_name"`
	Error           string    `json:"error,omitempty"`
}
