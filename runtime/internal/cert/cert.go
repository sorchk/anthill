package cert

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

type NodeCert struct {
	CertFile string
	KeyFile  string
	CACert   *x509.Certificate
	Cert     *x509.Certificate
	Key      *rsa.PrivateKey
}

func LoadFromFiles(certFile, keyFile, caCertFile string) (*NodeCert, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cert file: %w", err)
	}

	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	var caCert *x509.Certificate
	if caCertFile != "" {
		caCertPEM, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert file: %w", err)
		}
		caCert, err = ParseCertificate(caCertPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to parse CA cert: %w", err)
		}
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse X509 key pair: %w", err)
	}

	cert, err := ParseCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	key, ok := tlsCert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}

	return &NodeCert{
		CertFile: certFile,
		KeyFile:  keyFile,
		CACert:   caCert,
		Cert:     cert,
		Key:      key,
	}, nil
}

func ParseCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}
	return cert, nil
}

func Save(certPEM, keyPEM []byte, certFile, keyFile string) error {
	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		return fmt.Errorf("failed to write cert file: %w", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}
	return nil
}

func (nc *NodeCert) Verify() error {
	if nc.Cert == nil {
		return fmt.Errorf("certificate is nil")
	}

	opts := x509.VerifyOptions{
		Roots:         nil,
		Intermediates: x509.NewCertPool(),
	}

	if nc.CACert != nil {
		opts.Roots = x509.NewCertPool()
		opts.Roots.AddCert(nc.CACert)
	} else {
		opts.Roots = x509.NewCertPool()
	}

	_, err := nc.Cert.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}

func (nc *NodeCert) IsExpired() bool {
	if nc.Cert == nil {
		return true
	}
	now := time.Now()
	return now.Before(nc.Cert.NotBefore) || now.After(nc.Cert.NotAfter)
}

func (nc *NodeCert) GetCommonName() string {
	if nc.Cert == nil || nc.Cert.Subject.CommonName == "" {
		return ""
	}
	return nc.Cert.Subject.CommonName
}