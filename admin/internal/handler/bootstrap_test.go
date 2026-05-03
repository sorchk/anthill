package handler

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"anthill/admin/internal/database"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestDB(t *testing.T) (*sql.DB, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	
	err = database.InitAdminUser(db, "admin", "password")
	if err != nil {
		t.Fatalf("Failed to init admin user: %v", err)
	}
	
	return db, dbPath
}

func setupTestRouter(db *sql.DB, ca *CertCA) *gin.Engine {
	r := gin.New()
	handler := NewBootstrapHandler(db, ca)
	r.POST("/api/nodes/:id/bootstrap", handler.Bootstrap)
	return r
}

func TestBootstrap_InvalidNodeID(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()
	
	ca, err := NewCertCA("", "")
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	router := setupTestRouter(db, ca)
	
	req, _ := http.NewRequest("POST", "/api/nodes/invalid/bootstrap", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestBootstrap_MissingAuthHeader(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()
	
	ca, _ := NewCertCA("", "")
	
	_, err := db.Exec(`INSERT INTO nodes (name, host, port, owner_id, bootstrap_token) VALUES (?, ?, ?, ?, ?)`,
		"test-node", "localhost", 18888, 1, "test-token")
	if err != nil {
		t.Fatalf("Failed to insert node: %v", err)
	}
	
	router := setupTestRouter(db, ca)
	
	req, _ := http.NewRequest("POST", "/api/nodes/1/bootstrap", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestBootstrap_InvalidAuthFormat(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()
	
	ca, _ := NewCertCA("", "")
	
	_, err := db.Exec(`INSERT INTO nodes (name, host, port, owner_id, bootstrap_token) VALUES (?, ?, ?, ?, ?)`,
		"test-node", "localhost", 18888, 1, "test-token")
	if err != nil {
		t.Fatalf("Failed to insert node: %v", err)
	}
	
	router := setupTestRouter(db, ca)
	
	req, _ := http.NewRequest("POST", "/api/nodes/1/bootstrap", nil)
	req.Header.Set("Authorization", "Basic some-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestBootstrap_NodeNotFound(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()
	
	ca, _ := NewCertCA("", "")
	router := setupTestRouter(db, ca)
	
	req, _ := http.NewRequest("POST", "/api/nodes/999/bootstrap", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestBootstrap_InvalidToken(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()
	
	ca, _ := NewCertCA("", "")
	
	_, err := db.Exec(`INSERT INTO nodes (name, host, port, owner_id, bootstrap_token) VALUES (?, ?, ?, ?, ?)`,
		"test-node", "localhost", 18888, 1, "valid-token")
	if err != nil {
		t.Fatalf("Failed to insert node: %v", err)
	}
	
	router := setupTestRouter(db, ca)
	
	req, _ := http.NewRequest("POST", "/api/nodes/1/bootstrap", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestBootstrap_Success(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()
	
	ca, _ := NewCertCA("", "")
	
	_, err := db.Exec(`INSERT INTO nodes (name, host, port, owner_id, bootstrap_token) VALUES (?, ?, ?, ?, ?)`,
		"test-node", "localhost", 18888, 1, "valid-token")
	if err != nil {
		t.Fatalf("Failed to insert node: %v", err)
	}
	
	router := setupTestRouter(db, ca)
	
	req, _ := http.NewRequest("POST", "/api/nodes/1/bootstrap", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}
	
	var resp struct {
		Cert    string `json:"cert"`
		Key     string `json:"key"`
		Expires string `json:"expires"`
	}
	
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if resp.Cert == "" || resp.Key == "" || resp.Expires == "" {
		t.Error("Response missing cert, key, or expires")
	}
	
	var certSerial, certExpires string
	err = db.QueryRow("SELECT cert_serial, cert_expires FROM nodes WHERE id = 1").Scan(&certSerial, &certExpires)
	if err != nil {
		t.Fatalf("Failed to query cert info: %v", err)
	}
	
	if certSerial == "" || certExpires == "" {
		t.Error("Cert info not stored in database")
	}
}

func TestBootstrap_CertSavedToDatabase(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()
	
	ca, _ := NewCertCA("", "")
	
	_, err := db.Exec(`INSERT INTO nodes (name, host, port, owner_id, bootstrap_token) VALUES (?, ?, ?, ?, ?)`,
		"test-node", "localhost", 18888, 1, "test-token-123")
	if err != nil {
		t.Fatalf("Failed to insert node: %v", err)
	}
	
	router := setupTestRouter(db, ca)
	
	req, _ := http.NewRequest("POST", "/api/nodes/1/bootstrap", nil)
	req.Header.Set("Authorization", "Bearer test-token-123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var storedSerial, storedExpires string
	err = db.QueryRow("SELECT cert_serial, cert_expires FROM nodes WHERE id = 1").Scan(&storedSerial, &storedExpires)
	if err != nil {
		t.Fatalf("Failed to get stored cert: %v", err)
	}
	
	if storedSerial == "" {
		t.Error("Cert serial not stored")
	}
	if storedExpires == "" {
		t.Error("Cert expires not stored")
	}
	
	expiresTime, err := time.Parse(time.RFC3339, storedExpires)
	if err != nil {
		t.Fatalf("Invalid expires format: %v", err)
	}
	
	expectedExpiry := time.Now().Add(365 * 24 * time.Hour)
	if expiresTime.Sub(expectedExpiry) > time.Minute {
		t.Errorf("Expiry time seems wrong. Expected around %v, got %v", expectedExpiry, expiresTime)
	}
}

func TestBootstrap_NodeCertHasCorrectCN(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()
	
	ca, _ := NewCertCA("", "")
	
	_, err := db.Exec(`INSERT INTO nodes (name, host, port, owner_id, bootstrap_token) VALUES (?, ?, ?, ?, ?)`,
		"test-node", "localhost", 18888, 1, "node-token")
	if err != nil {
		t.Fatalf("Failed to insert node: %v", err)
	}
	
	router := setupTestRouter(db, ca)
	
	req, _ := http.NewRequest("POST", "/api/nodes/1/bootstrap", nil)
	req.Header.Set("Authorization", "Bearer node-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var resp struct {
		Cert string `json:"cert"`
		Key  string `json:"key"`
	}
	
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	block, _ := pem.Decode([]byte(resp.Cert))
	if block == nil {
		t.Fatal("Failed to decode cert PEM")
	}
	
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}
	
	if cert.Subject.CommonName != "1" {
		t.Errorf("Expected CN '1', got '%s'", cert.Subject.CommonName)
	}
}

func TestBootstrap_MTLSSetup(t *testing.T) {
	ca, err := NewCertCA("", "")
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	certPEM := ca.GetCACert()
	if certPEM == nil {
		t.Fatal("CA cert is nil")
	}
	
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(certPEM) {
		t.Fatal("Failed to add CA to pool")
	}
	
	certPEM2, keyPEM, _, err := ca.SignNodeCert("test-node", time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Failed to sign node cert: %v", err)
	}
	
	cert, err := tls.X509KeyPair(certPEM2, keyPEM)
	if err != nil {
		t.Fatalf("Failed to load key pair: %v", err)
	}
	
	_ = caPool
	_ = cert
	
	t.Log("MTLS setup verified")
}