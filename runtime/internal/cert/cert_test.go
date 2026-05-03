package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestLoadFromFiles(t *testing.T) {
	tmpDir := t.TempDir()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{CommonName: "test-node"},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{CommonName: "Test CA"},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageCertSign,
		IsCA:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, caTemplate, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	certFile := tmpDir + "/cert.pem"
	keyFile := tmpDir + "/key.pem"
	caCertFile := tmpDir + "/ca.pem"

	if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caCertFile, caCertPEM, 0600); err != nil {
		t.Fatal(err)
	}

	nc, err := LoadFromFiles(certFile, keyFile, caCertFile)
	if err != nil {
		t.Fatalf("LoadFromFiles failed: %v", err)
	}

	if nc.GetCommonName() != "test-node" {
		t.Errorf("expected CN 'test-node', got '%s'", nc.GetCommonName())
	}

	if nc.IsExpired() {
		t.Error("certificate should not be expired")
	}

	if err := nc.Verify(); err != nil {
		t.Errorf("Verify failed: %v", err)
	}
}

func TestIsExpired(t *testing.T) {
	tmpDir := t.TempDir()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "expired-test"},
		NotBefore:    time.Now().Add(-48 * time.Hour),
		NotAfter:     time.Now().Add(-24 * time.Hour),
	}

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	certFile := tmpDir + "/cert.pem"
	keyFile := tmpDir + "/key.pem"

	os.WriteFile(certFile, certPEM, 0600)
	os.WriteFile(keyFile, keyPEM, 0600)

	nc, err := LoadFromFiles(certFile, keyFile, "")
	if err != nil {
		t.Fatalf("LoadFromFiles failed: %v", err)
	}

	if !nc.IsExpired() {
		t.Error("expired certificate should return true")
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()

	certPEM := []byte("test-cert-pem")
	keyPEM := []byte("test-key-pem")

	certFile := tmpDir + "/new-cert.pem"
	keyFile := tmpDir + "/new-key.pem"

	err := Save(certPEM, keyPEM, certFile, keyFile)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	savedCert, err := os.ReadFile(certFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(savedCert) != string(certPEM) {
		t.Error("saved cert does not match")
	}

	savedKey, err := os.ReadFile(keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(savedKey) != string(keyPEM) {
		t.Error("saved key does not match")
	}
}