package pki

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"
)

// CertificateRecord represents a certificate stored in the database
type CertificateRecord struct {
	ID           string
	Type         string // "root_ca", "server_ca", "agent_ca", "server", "agent"
	CommonName   string
	SerialNumber string
	Certificate  []byte // PEM-encoded certificate
	PrivateKey   []byte // PEM-encoded private key (encrypted)
	CAChain      []byte // PEM-encoded CA certificate chain
	IssuedAt     time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
	NodeID       string // For agent certificates
}

// Storage defines the interface for certificate persistence
type Storage interface {
	// StoreCertificate stores a certificate record
	StoreCertificate(ctx context.Context, record *CertificateRecord) error

	// GetCertificate retrieves a certificate by ID
	GetCertificate(ctx context.Context, id string) (*CertificateRecord, error)

	// GetCertificateBySerial retrieves a certificate by serial number
	GetCertificateBySerial(ctx context.Context, serialNumber string) (*CertificateRecord, error)

	// ListCertificates lists all certificates of a given type
	ListCertificates(ctx context.Context, certType string) ([]*CertificateRecord, error)

	// ListExpiringCertificates lists certificates expiring within the given duration
	ListExpiringCertificates(ctx context.Context, within time.Duration) ([]*CertificateRecord, error)

	// RevokeCertificate marks a certificate as revoked
	RevokeCertificate(ctx context.Context, id string) error

	// GetRevokedCertificates retrieves all revoked certificates
	GetRevokedCertificates(ctx context.Context) ([]*CertificateRecord, error)
}

// FileStorage implements certificate storage using the filesystem
type FileStorage struct {
	basePath string
}

// NewFileStorage creates a new file-based certificate storage
func NewFileStorage(basePath string) *FileStorage {
	return &FileStorage{
		basePath: basePath,
	}
}

// StoreCertificate stores a certificate to the filesystem
func (s *FileStorage) StoreCertificate(ctx context.Context, record *CertificateRecord) error {
	// Implementation would write to files
	// For now, this is a placeholder
	return fmt.Errorf("file storage not yet implemented")
}

// GetCertificate retrieves a certificate from the filesystem
func (s *FileStorage) GetCertificate(ctx context.Context, id string) (*CertificateRecord, error) {
	return nil, fmt.Errorf("file storage not yet implemented")
}

// GetCertificateBySerial retrieves a certificate by serial number
func (s *FileStorage) GetCertificateBySerial(ctx context.Context, serialNumber string) (*CertificateRecord, error) {
	return nil, fmt.Errorf("file storage not yet implemented")
}

// ListCertificates lists all certificates of a given type
func (s *FileStorage) ListCertificates(ctx context.Context, certType string) ([]*CertificateRecord, error) {
	return nil, fmt.Errorf("file storage not yet implemented")
}

// ListExpiringCertificates lists certificates expiring within the given duration
func (s *FileStorage) ListExpiringCertificates(ctx context.Context, within time.Duration) ([]*CertificateRecord, error) {
	return nil, fmt.Errorf("file storage not yet implemented")
}

// RevokeCertificate marks a certificate as revoked
func (s *FileStorage) RevokeCertificate(ctx context.Context, id string) error {
	return fmt.Errorf("file storage not yet implemented")
}

// GetRevokedCertificates retrieves all revoked certificates
func (s *FileStorage) GetRevokedCertificates(ctx context.Context) ([]*CertificateRecord, error) {
	return nil, fmt.Errorf("file storage not yet implemented")
}

// ManagerWithStorage wraps a PKI manager with certificate storage
type ManagerWithStorage struct {
	*Manager
	storage Storage
}

// NewManagerWithStorage creates a PKI manager with storage
func NewManagerWithStorage(config *Config, storage Storage) *ManagerWithStorage {
	return &ManagerWithStorage{
		Manager: NewManager(config),
		storage: storage,
	}
}

// StoreRootCA stores the root CA to storage
func (m *ManagerWithStorage) StoreRootCA(ctx context.Context) error {
	if m.rootCA == nil {
		return fmt.Errorf("root CA not initialized")
	}

	record := &CertificateRecord{
		ID:           "root-ca",
		Type:         "root_ca",
		CommonName:   m.rootCA.Certificate.Subject.CommonName,
		SerialNumber: m.rootCA.Certificate.SerialNumber.String(),
		Certificate:  EncodeCertificatePEM(m.rootCA.Certificate),
		PrivateKey:   EncodePrivateKeyPEM(m.rootCA.PrivateKey),
		IssuedAt:     m.rootCA.Certificate.NotBefore,
		ExpiresAt:    m.rootCA.Certificate.NotAfter,
	}

	return m.storage.StoreCertificate(ctx, record)
}

// StoreIntermediateCAs stores the intermediate CAs to storage
func (m *ManagerWithStorage) StoreIntermediateCAs(ctx context.Context) error {
	// Store server CA
	if m.serverCA != nil {
		record := &CertificateRecord{
			ID:           "server-ca",
			Type:         "server_ca",
			CommonName:   m.serverCA.Certificate.Subject.CommonName,
			SerialNumber: m.serverCA.Certificate.SerialNumber.String(),
			Certificate:  EncodeCertificatePEM(m.serverCA.Certificate),
			PrivateKey:   EncodePrivateKeyPEM(m.serverCA.PrivateKey),
			CAChain:      EncodeCertificatePEM(m.rootCA.Certificate),
			IssuedAt:     m.serverCA.Certificate.NotBefore,
			ExpiresAt:    m.serverCA.Certificate.NotAfter,
		}
		if err := m.storage.StoreCertificate(ctx, record); err != nil {
			return fmt.Errorf("failed to store server CA: %w", err)
		}
	}

	// Store agent CA
	if m.agentCA != nil {
		record := &CertificateRecord{
			ID:           "agent-ca",
			Type:         "agent_ca",
			CommonName:   m.agentCA.Certificate.Subject.CommonName,
			SerialNumber: m.agentCA.Certificate.SerialNumber.String(),
			Certificate:  EncodeCertificatePEM(m.agentCA.Certificate),
			PrivateKey:   EncodePrivateKeyPEM(m.agentCA.PrivateKey),
			CAChain:      EncodeCertificatePEM(m.rootCA.Certificate),
			IssuedAt:     m.agentCA.Certificate.NotBefore,
			ExpiresAt:    m.agentCA.Certificate.NotAfter,
		}
		if err := m.storage.StoreCertificate(ctx, record); err != nil {
			return fmt.Errorf("failed to store agent CA: %w", err)
		}
	}

	return nil
}

// StoreAgentCertificate stores an agent certificate to storage
func (m *ManagerWithStorage) StoreAgentCertificate(ctx context.Context, cert *Certificate, agentID, nodeID string) error {
	// Create CA chain (agent CA + root CA)
	caChain := append(EncodeCertificatePEM(m.agentCA.Certificate), EncodeCertificatePEM(m.rootCA.Certificate)...)

	record := &CertificateRecord{
		ID:           agentID,
		Type:         "agent",
		CommonName:   cert.Certificate.Subject.CommonName,
		SerialNumber: cert.Certificate.SerialNumber.String(),
		Certificate:  EncodeCertificatePEM(cert.Certificate),
		PrivateKey:   EncodePrivateKeyPEM(cert.PrivateKey),
		CAChain:      caChain,
		IssuedAt:     cert.Certificate.NotBefore,
		ExpiresAt:    cert.Certificate.NotAfter,
		NodeID:       nodeID,
	}

	return m.storage.StoreCertificate(ctx, record)
}

// StoreServerCertificate stores a server certificate to storage
func (m *ManagerWithStorage) StoreServerCertificate(ctx context.Context, cert *Certificate, serverID string) error {
	// Create CA chain (server CA + root CA)
	caChain := append(EncodeCertificatePEM(m.serverCA.Certificate), EncodeCertificatePEM(m.rootCA.Certificate)...)

	record := &CertificateRecord{
		ID:           serverID,
		Type:         "server",
		CommonName:   cert.Certificate.Subject.CommonName,
		SerialNumber: cert.Certificate.SerialNumber.String(),
		Certificate:  EncodeCertificatePEM(cert.Certificate),
		PrivateKey:   EncodePrivateKeyPEM(cert.PrivateKey),
		CAChain:      caChain,
		IssuedAt:     cert.Certificate.NotBefore,
		ExpiresAt:    cert.Certificate.NotAfter,
	}

	return m.storage.StoreCertificate(ctx, record)
}

// GenerateCRL generates a Certificate Revocation List
func (m *ManagerWithStorage) GenerateCRL(ctx context.Context) ([]byte, error) {
	// Get all revoked certificates
	revokedCerts, err := m.storage.GetRevokedCertificates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get revoked certificates: %w", err)
	}

	// Build revoked certificate list
	var revokedList []pkix.RevokedCertificate
	for _, cert := range revokedCerts {
		// Parse certificate to get serial number
		x509Cert, err := DecodeCertificatePEM(cert.Certificate)
		if err != nil {
			continue
		}

		revokedList = append(revokedList, pkix.RevokedCertificate{
			SerialNumber:   x509Cert.SerialNumber,
			RevocationTime: *cert.RevokedAt,
		})
	}

	// Create CRL template
	template := &x509.RevocationList{
		Number:              big.NewInt(time.Now().Unix()),
		ThisUpdate:          time.Now(),
		NextUpdate:          time.Now().Add(7 * 24 * time.Hour), // 7 days
		RevokedCertificates: revokedList,
	}

	// Sign CRL with root CA
	crlDER, err := x509.CreateRevocationList(rand.Reader, template, m.rootCA.Certificate, m.rootCA.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CRL: %w", err)
	}

	return crlDER, nil
}
