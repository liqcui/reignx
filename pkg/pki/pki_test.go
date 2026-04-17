package pki

import (
	"crypto/x509"
	"testing"
	"time"
)

func TestPKIInitialization(t *testing.T) {
	config := DefaultConfig()
	manager := NewManager(config)

	// Initialize root CA
	if err := manager.InitializeRootCA(); err != nil {
		t.Fatalf("Failed to initialize root CA: %v", err)
	}

	rootCA := manager.GetRootCA()
	if rootCA == nil {
		t.Fatal("Root CA is nil")
	}

	if rootCA.Certificate == nil {
		t.Fatal("Root CA certificate is nil")
	}

	if rootCA.PrivateKey == nil {
		t.Fatal("Root CA private key is nil")
	}

	// Verify root CA properties
	if !rootCA.Certificate.IsCA {
		t.Error("Root CA certificate should be a CA")
	}

	if rootCA.Certificate.Subject.CommonName != "ReignX Root CA" {
		t.Errorf("Expected CN 'ReignX Root CA', got '%s'", rootCA.Certificate.Subject.CommonName)
	}

	t.Logf("Root CA initialized: CN=%s, Valid until=%s",
		rootCA.Certificate.Subject.CommonName,
		rootCA.Certificate.NotAfter.Format(time.RFC3339))
}

func TestIntermediateCACreation(t *testing.T) {
	manager := NewManager(nil)

	// Initialize root CA first
	if err := manager.InitializeRootCA(); err != nil {
		t.Fatalf("Failed to initialize root CA: %v", err)
	}

	// Initialize intermediate CAs
	if err := manager.InitializeIntermediateCA(); err != nil {
		t.Fatalf("Failed to initialize intermediate CAs: %v", err)
	}

	serverCA := manager.GetServerCA()
	if serverCA == nil {
		t.Fatal("Server CA is nil")
	}

	agentCA := manager.GetAgentCA()
	if agentCA == nil {
		t.Fatal("Agent CA is nil")
	}

	// Verify server CA
	if !serverCA.Certificate.IsCA {
		t.Error("Server CA certificate should be a CA")
	}

	if serverCA.Certificate.Subject.CommonName != "ReignX Server Intermediate CA" {
		t.Errorf("Expected CN 'ReignX Server Intermediate CA', got '%s'",
			serverCA.Certificate.Subject.CommonName)
	}

	// Verify agent CA
	if !agentCA.Certificate.IsCA {
		t.Error("Agent CA certificate should be a CA")
	}

	if agentCA.Certificate.Subject.CommonName != "ReignX Agent Intermediate CA" {
		t.Errorf("Expected CN 'ReignX Agent Intermediate CA', got '%s'",
			agentCA.Certificate.Subject.CommonName)
	}

	t.Logf("Server CA: CN=%s, Valid until=%s",
		serverCA.Certificate.Subject.CommonName,
		serverCA.Certificate.NotAfter.Format(time.RFC3339))

	t.Logf("Agent CA: CN=%s, Valid until=%s",
		agentCA.Certificate.Subject.CommonName,
		agentCA.Certificate.NotAfter.Format(time.RFC3339))
}

func TestServerCertificateGeneration(t *testing.T) {
	manager := NewManager(nil)

	// Initialize PKI
	if err := manager.InitializeRootCA(); err != nil {
		t.Fatalf("Failed to initialize root CA: %v", err)
	}
	if err := manager.InitializeIntermediateCA(); err != nil {
		t.Fatalf("Failed to initialize intermediate CAs: %v", err)
	}

	// Generate server certificate
	dnsNames := []string{"localhost", "reignx-apiserver", "reignx-apiserver.default.svc.cluster.local"}
	ipAddresses := []string{"127.0.0.1", "::1"}

	cert, err := manager.GenerateServerCertificate("reignx-apiserver", dnsNames, ipAddresses)
	if err != nil {
		t.Fatalf("Failed to generate server certificate: %v", err)
	}

	if cert == nil {
		t.Fatal("Generated certificate is nil")
	}

	if cert.Certificate == nil {
		t.Fatal("Certificate is nil")
	}

	if cert.PrivateKey == nil {
		t.Fatal("Private key is nil")
	}

	// Verify certificate properties
	if cert.Certificate.IsCA {
		t.Error("Server certificate should not be a CA")
	}

	if cert.Certificate.Subject.CommonName != "reignx-apiserver" {
		t.Errorf("Expected CN 'reignx-apiserver', got '%s'",
			cert.Certificate.Subject.CommonName)
	}

	// Verify DNS names
	if len(cert.Certificate.DNSNames) != len(dnsNames) {
		t.Errorf("Expected %d DNS names, got %d", len(dnsNames), len(cert.Certificate.DNSNames))
	}

	// Verify extended key usage
	hasServerAuth := false
	for _, usage := range cert.Certificate.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
			break
		}
	}
	if !hasServerAuth {
		t.Error("Server certificate should have ExtKeyUsageServerAuth")
	}

	t.Logf("Server certificate: CN=%s, DNS names=%v, Valid until=%s",
		cert.Certificate.Subject.CommonName,
		cert.Certificate.DNSNames,
		cert.Certificate.NotAfter.Format(time.RFC3339))
}

func TestAgentCertificateGeneration(t *testing.T) {
	manager := NewManager(nil)

	// Initialize PKI
	if err := manager.InitializeRootCA(); err != nil {
		t.Fatalf("Failed to initialize root CA: %v", err)
	}
	if err := manager.InitializeIntermediateCA(); err != nil {
		t.Fatalf("Failed to initialize intermediate CAs: %v", err)
	}

	// Generate agent certificate
	agentID := "agent-12345"
	nodeID := "node-67890"

	cert, err := manager.GenerateAgentCertificate(agentID, nodeID)
	if err != nil {
		t.Fatalf("Failed to generate agent certificate: %v", err)
	}

	if cert == nil {
		t.Fatal("Generated certificate is nil")
	}

	// Verify certificate properties
	if cert.Certificate.IsCA {
		t.Error("Agent certificate should not be a CA")
	}

	expectedCN := "agent-" + agentID
	if cert.Certificate.Subject.CommonName != expectedCN {
		t.Errorf("Expected CN '%s', got '%s'", expectedCN, cert.Certificate.Subject.CommonName)
	}

	// Verify extended key usage
	hasClientAuth := false
	for _, usage := range cert.Certificate.ExtKeyUsage {
		if usage == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
			break
		}
	}
	if !hasClientAuth {
		t.Error("Agent certificate should have ExtKeyUsageClientAuth")
	}

	t.Logf("Agent certificate: CN=%s, Valid until=%s",
		cert.Certificate.Subject.CommonName,
		cert.Certificate.NotAfter.Format(time.RFC3339))
}

func TestCertificatePEMEncoding(t *testing.T) {
	manager := NewManager(nil)

	// Initialize PKI and generate a certificate
	if err := manager.InitializeRootCA(); err != nil {
		t.Fatalf("Failed to initialize root CA: %v", err)
	}
	if err := manager.InitializeIntermediateCA(); err != nil {
		t.Fatalf("Failed to initialize intermediate CAs: %v", err)
	}

	cert, err := manager.GenerateAgentCertificate("test-agent", "test-node")
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	// Encode certificate to PEM
	certPEM := EncodeCertificatePEM(cert.Certificate)
	if len(certPEM) == 0 {
		t.Fatal("PEM-encoded certificate is empty")
	}

	// Encode private key to PEM
	keyPEM := EncodePrivateKeyPEM(cert.PrivateKey)
	if len(keyPEM) == 0 {
		t.Fatal("PEM-encoded private key is empty")
	}

	// Decode certificate from PEM
	decodedCert, err := DecodeCertificatePEM(certPEM)
	if err != nil {
		t.Fatalf("Failed to decode certificate PEM: %v", err)
	}

	if decodedCert.Subject.CommonName != cert.Certificate.Subject.CommonName {
		t.Error("Decoded certificate CN does not match original")
	}

	// Decode private key from PEM
	decodedKey, err := DecodePrivateKeyPEM(keyPEM)
	if err != nil {
		t.Fatalf("Failed to decode private key PEM: %v", err)
	}

	if decodedKey.N.Cmp(cert.PrivateKey.N) != 0 {
		t.Error("Decoded private key does not match original")
	}

	t.Log("PEM encoding/decoding successful")
}

func TestCertificateRenewalCheck(t *testing.T) {
	manager := NewManager(nil)

	// Initialize PKI
	if err := manager.InitializeRootCA(); err != nil {
		t.Fatalf("Failed to initialize root CA: %v", err)
	}
	if err := manager.InitializeIntermediateCA(); err != nil {
		t.Fatalf("Failed to initialize intermediate CAs: %v", err)
	}

	// Generate a fresh certificate
	cert, err := manager.GenerateAgentCertificate("test-agent", "test-node")
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	// Check if renewal is needed (should not be needed for fresh cert)
	if NeedRenewal(cert.Certificate) {
		t.Error("Fresh certificate should not need renewal")
	}

	// Check if expired (should not be expired)
	if IsExpired(cert.Certificate) {
		t.Error("Fresh certificate should not be expired")
	}

	t.Logf("Certificate valid for %v", time.Until(cert.Certificate.NotAfter))
}

func TestCertificateChainVerification(t *testing.T) {
	manager := NewManager(nil)

	// Initialize PKI
	if err := manager.InitializeRootCA(); err != nil {
		t.Fatalf("Failed to initialize root CA: %v", err)
	}
	if err := manager.InitializeIntermediateCA(); err != nil {
		t.Fatalf("Failed to initialize intermediate CAs: %v", err)
	}

	// Generate agent certificate
	cert, err := manager.GenerateAgentCertificate("test-agent", "test-node")
	if err != nil {
		t.Fatalf("Failed to generate certificate: %v", err)
	}

	// Create certificate pool with root CA
	roots := x509.NewCertPool()
	roots.AddCert(manager.GetRootCA().Certificate)

	// Create intermediate pool
	intermediates := x509.NewCertPool()
	intermediates.AddCert(manager.GetAgentCA().Certificate)

	// Verify certificate chain
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	chains, err := cert.Certificate.Verify(opts)
	if err != nil {
		t.Fatalf("Failed to verify certificate chain: %v", err)
	}

	if len(chains) == 0 {
		t.Fatal("No valid certificate chains found")
	}

	t.Logf("Certificate chain verified: %d chain(s) found", len(chains))
	for i, chain := range chains {
		t.Logf("Chain %d:", i+1)
		for j, cert := range chain {
			t.Logf("  [%d] CN=%s, IsCA=%v", j, cert.Subject.CommonName, cert.IsCA)
		}
	}
}
