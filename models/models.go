package models

import "database/sql"

// DomainInfo represents the relevant data we want to extract from the stream
type DomainInfo struct {
	ID                  int64  `json:"-"` // not returned in JSON
	Domain              string `json:"domain"`
	IsApex              bool   `json:"is_apex"`
	ParentDomain        string `json:"parent_domain"`
	NotBefore           int64  `json:"-"`
	NotBeforeTime       string `json:"not_before"`
	NotAfter            int64  `json:"-"` // not returned in JSON
	NotAfterTime        string `json:"-"` // not returned in JSON
	SerialNumber        string `json:"serial_number"`
	Fingerprint         string `json:"fingerprint"`
	KeyUsage            string `json:"key_usage"`
	ExtendedKeyUsage    string `json:"extended_key_usage"`
	SubjectKeyID        string `json:"subject_key_id"`
	AuthorityKeyID      string `json:"authority_key_id"`
	AuthorityInfo       string `json:"authority_info"`
	SubjectAltName      string `json:"subject_alt_name"`
	CertificatePolicies string `json:"certificate_policies"`
	Wildcard            bool   `json:"wildcard"`
}

type DomainWithSubdomains struct {
	Domain     string   `json:"domain"`
	Subdomains []string `json:"subdomains"`
}

type Server struct {
	Db *sql.DB
}
