package pki

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// CertificateAuthority represents a Certificate Authority
type CertificateAuthority struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
}

// Config contains PKI configuration
type Config struct {
	Organization       string
	Country            string
	Province           string
	Locality           string
	RootCAValidityDays int
	IntCAValidityDays  int
	CertValidityDays   int
	KeySize            int
}

// DefaultConfig returns default PKI configuration
func DefaultConfig() *Config {
	return &Config{
		Organization:       "ReignX",
		Country:            "US",
		Province:           "California",
		Locality:           "San Francisco",
		RootCAValidityDays: 3650, // 10 years
		IntCAValidityDays:  1825, // 5 years
		CertValidityDays:   365,  // 1 year
		KeySize:            2048,
	}
}

// Manager handles PKI operations
type Manager struct {
	config   *Config
	rootCA   *CertificateAuthority
	serverCA *CertificateAuthority
	agentCA  *CertificateAuthority
}

// NewManager creates a new PKI manager
func NewManager(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	}
	return &Manager{
		config: config,
	}
}

// InitializeRootCA creates or loads the root CA
func (m *Manager) InitializeRootCA() error {
	// Generate root CA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, m.config.KeySize)
	if err != nil {
		return fmt.Errorf("failed to generate root CA private key: %w", err)
	}

	// Create root CA certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{m.config.Organization},
			Country:      []string{m.config.Country},
			Province:     []string{m.config.Province},
			Locality:     []string{m.config.Locality},
			CommonName:   "ReignX Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, m.config.RootCAValidityDays),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
	}

	// Self-sign the root CA certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create root CA certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("failed to parse root CA certificate: %w", err)
	}

	m.rootCA = &CertificateAuthority{
		Certificate: cert,
		PrivateKey:  privateKey,
	}

	return nil
}

// InitializeIntermediateCA creates intermediate CAs for servers and agents
func (m *Manager) InitializeIntermediateCA() error {
	if m.rootCA == nil {
		return fmt.Errorf("root CA must be initialized first")
	}

	// Create server intermediate CA
	serverCA, err := m.createIntermediateCA("ReignX Server Intermediate CA", 1)
	if err != nil {
		return fmt.Errorf("failed to create server intermediate CA: %w", err)
	}
	m.serverCA = serverCA

	// Create agent intermediate CA
	agentCA, err := m.createIntermediateCA("ReignX Agent Intermediate CA", 2)
	if err != nil {
		return fmt.Errorf("failed to create agent intermediate CA: %w", err)
	}
	m.agentCA = agentCA

	return nil
}

// createIntermediateCA creates an intermediate CA
func (m *Manager) createIntermediateCA(commonName string, serialNumber int64) (*CertificateAuthority, error) {
	// Generate intermediate CA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, m.config.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate intermediate CA private key: %w", err)
	}

	// Create intermediate CA certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serialNumber),
		Subject: pkix.Name{
			Organization: []string{m.config.Organization},
			Country:      []string{m.config.Country},
			Province:     []string{m.config.Province},
			Locality:     []string{m.config.Locality},
			CommonName:   commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, m.config.IntCAValidityDays),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Sign with root CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, m.rootCA.Certificate, &privateKey.PublicKey, m.rootCA.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create intermediate CA certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intermediate CA certificate: %w", err)
	}

	return &CertificateAuthority{
		Certificate: cert,
		PrivateKey:  privateKey,
	}, nil
}

// GenerateServerCertificate creates a certificate for a server
func (m *Manager) GenerateServerCertificate(commonName string, dnsNames []string, ipAddresses []string) (*Certificate, error) {
	if m.serverCA == nil {
		return nil, fmt.Errorf("server CA must be initialized first")
	}

	return m.generateCertificate(m.serverCA, commonName, dnsNames, ipAddresses, false)
}

// GenerateAgentCertificate creates a certificate for an agent
func (m *Manager) GenerateAgentCertificate(agentID string, nodeID string) (*Certificate, error) {
	if m.agentCA == nil {
		return nil, fmt.Errorf("agent CA must be initialized first")
	}

	commonName := fmt.Sprintf("agent-%s", agentID)
	return m.generateCertificate(m.agentCA, commonName, nil, nil, true)
}

// generateCertificate creates a certificate signed by the given CA
func (m *Manager) generateCertificate(ca *CertificateAuthority, commonName string, dnsNames []string, ipAddresses []string, isClient bool) (*Certificate, error) {
	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, m.config.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Determine key usage and extended key usage
	keyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	if isClient {
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{m.config.Organization},
			Country:      []string{m.config.Country},
			Province:     []string{m.config.Province},
			Locality:     []string{m.config.Locality},
			CommonName:   commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, m.config.CertValidityDays),
		KeyUsage:              keyUsage,
		ExtKeyUsage:           extKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Add DNS names and IP addresses if provided
	if len(dnsNames) > 0 {
		for _, dns := range dnsNames {
			template.DNSNames = append(template.DNSNames, dns)
		}
	}

	// Sign the certificate with the CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.Certificate, &privateKey.PublicKey, ca.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return &Certificate{
		Certificate: cert,
		PrivateKey:  privateKey,
		CA:          ca.Certificate,
	}, nil
}

// Certificate represents a certificate with its private key
type Certificate struct {
	Certificate *x509.Certificate
	PrivateKey  *rsa.PrivateKey
	CA          *x509.Certificate
}

// EncodeCertificatePEM encodes a certificate to PEM format
func EncodeCertificatePEM(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
}

// EncodePrivateKeyPEM encodes a private key to PEM format
func EncodePrivateKeyPEM(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// DecodeCertificatePEM decodes a PEM-encoded certificate
func DecodeCertificatePEM(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}

// DecodePrivateKeyPEM decodes a PEM-encoded private key
func DecodePrivateKeyPEM(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return key, nil
}

// NeedRenewal checks if a certificate needs renewal (within 30 days)
func NeedRenewal(cert *x509.Certificate) bool {
	renewalThreshold := 30 * 24 * time.Hour
	return time.Until(cert.NotAfter) < renewalThreshold
}

// IsExpired checks if a certificate is expired
func IsExpired(cert *x509.Certificate) bool {
	return time.Now().After(cert.NotAfter)
}

// GetRootCA returns the root CA
func (m *Manager) GetRootCA() *CertificateAuthority {
	return m.rootCA
}

// GetServerCA returns the server intermediate CA
func (m *Manager) GetServerCA() *CertificateAuthority {
	return m.serverCA
}

// GetAgentCA returns the agent intermediate CA
func (m *Manager) GetAgentCA() *CertificateAuthority {
	return m.agentCA
}
