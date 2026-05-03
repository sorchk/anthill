package handler

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/google/uuid"
)

type CertCA struct {
	caCert *tls.Certificate
}

func NewCertCA(certFile, keyFile string) (*CertCA, error) {
	ca := &CertCA{}

	if certFile != "" && keyFile != "" {
		certPEM, err := os.ReadFile(certFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}

		keyPEM, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA key: %w", err)
		}

		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CA cert/key: %w", err)
		}

		ca.caCert = &cert
	} else {
		caCert, err := ca.generateCACert()
		if err != nil {
			return nil, fmt.Errorf("failed to generate CA cert: %w", err)
		}
		ca.caCert = caCert
	}

	return ca, nil
}

func (ca *CertCA) generateCACert() (*tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "Anthill CA",
			Organization: []string{"Anthill"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

func (ca *CertCA) GetCACert() []byte {
	if ca.caCert == nil {
		return nil
	}

	for _, cert := range ca.caCert.Certificate {
		certBlock := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert,
		}
		return pem.EncodeToMemory(certBlock)
	}
	return nil
}

func (ca *CertCA) GetCACertTLS() *tls.Certificate {
	return ca.caCert
}

func (ca *CertCA) GetCACertX509() *x509.Certificate {
	if ca.caCert == nil || len(ca.caCert.Certificate) == 0 {
		return nil
	}
	cert, err := x509.ParseCertificate(ca.caCert.Certificate[0])
	if err != nil {
		return nil
	}
	return cert
}

func (ca *CertCA) SignNodeCert(nodeID string, expires time.Time) (certPEM, keyPEM []byte, serial string, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate private key: %w", err)
	}

	serialBytes := make([]byte, 16)
	_, err = rand.Read(serialBytes)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate serial number: %w", err)
	}
	serialNumber := new(big.Int).SetBytes(serialBytes)

	serial = serialNumber.String()

	caCert, err := x509.ParseCertificate(ca.caCert.Certificate[0])
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to parse CA cert: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: nodeID,
		},
		NotBefore:   time.Now(),
		NotAfter:     expires,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ExtraExtensions: []pkix.Extension{
			{
				Id:       []int{1, 3, 6, 1, 5, 5, 7, 3, 2},
				Critical: false,
				Value:    []byte{0x30, 0x03, 0x01, 0x05},
			},
		},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &privateKey.PublicKey, ca.caCert.PrivateKey)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create certificate: %w", err)
	}

	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, serial, nil
}

func (ca *CertCA) ValidateToken(token string) bool {
	_, err := uuid.Parse(token)
	return err == nil
}

func (ca *CertCA) RenewNodeCert(nodeID string, expires time.Time) (certPEM, keyPEM []byte, serial string, err error) {
	return ca.SignNodeCert(nodeID, expires)
}

func (ca *CertCA) RevokeCert(serial string) error {
	return nil
}

func (ca *CertCA) GenerateCRL() ([]byte, error) {
	caCert, err := x509.ParseCertificate(ca.caCert.Certificate[0])
	if err != nil {
		return nil, err
	}

	signer, ok := ca.caCert.PrivateKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("CA private key does not implement crypto.Signer")
	}

	crlTemplate := &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: time.Now(),
		NextUpdate: time.Now().Add(24 * time.Hour),
	}

	crlDER, err := x509.CreateRevocationList(rand.Reader, crlTemplate, caCert, signer)
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "X509 CRL",
		Bytes: crlDER,
	}), nil
}