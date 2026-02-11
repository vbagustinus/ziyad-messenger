package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

// CertificateAuthority handles CA operations.
type CertificateAuthority struct {
	RootKey  *ecdsa.PrivateKey
	RootCert *x509.Certificate
}

// NewCertificateAuthority initializes a CA.
// In production, this would load from secure storage / HSM.
func NewCertificateAuthority() (*CertificateAuthority, error) {
	// 1. Generate Private Key (P-256)
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// 2. Create Root Certificate Template
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Secure LAN Chat Corp"},
			CommonName:   "Secure LAN Chat Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 Years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// 3. Self-sign the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create root certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated certificate: %w", err)
	}

	return &CertificateAuthority{
		RootKey:  privateKey,
		RootCert: cert,
	}, nil
}

// SaveToDisk persists the CA to disk (simulated secure storage).
func (ca *CertificateAuthority) SaveToDisk(certPath, keyPath string) error {
	// Save Certificate
	certOut, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to open cert.pem for writing: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: ca.RootCert.Raw}); err != nil {
		return fmt.Errorf("failed to write data to cert.pem: %w", err)
	}

	// Save Key
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open key.pem for writing: %w", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalECPrivateKey(ca.RootKey)
	if err != nil {
		return fmt.Errorf("unable to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write data to key.pem: %w", err)
	}

	return nil
}

func main() {
	fmt.Println("Initializing Root CA...")
	ca, err := NewCertificateAuthority()
	if err != nil {
		fmt.Printf("Error creating CA: %v\n", err)
		os.Exit(1)
	}

	// Ensure output directory exists
	if err := os.MkdirAll("certs", 0700); err != nil { // Secure permission
		fmt.Printf("Error creating certs directory: %v\n", err)
		os.Exit(1)
	}

	if err := ca.SaveToDisk("certs/root_ca.crt", "certs/root_ca.key"); err != nil {
		fmt.Printf("Error saving CA: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Root CA created and saved to certs/root_ca.crt and certs/root_ca.key")
}

// SignCSR signs a Certificate Signing Request and returns the certificate.
func (ca *CertificateAuthority) SignCSR(csrBytes []byte, validDuration time.Duration) ([]byte, error) {
	csr, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSR: %w", err)
	}

	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR signature check failed: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(validDuration),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, ca.RootCert, csr.PublicKey, ca.RootKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign certificate: %w", err)
	}

	return certBytes, nil
}
