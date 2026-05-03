package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"anthill-runtime/internal/cert"
)

type BootstrapRequest struct {
	Token string `json:"token"`
}

type BootstrapResponse struct {
	Cert    string `json:"cert"`
	Key     string `json:"key"`
	CACert  string `json:"ca_cert"`
	Expires string `json:"expires"`
}

type Bootstrapper struct {
	adminURL   string
	nodeID     string
	token      string
	certLoader *cert.NodeCert
}

func NewBootstrapper(adminURL, nodeID, token string, certLoader *cert.NodeCert) *Bootstrapper {
	return &Bootstrapper{
		adminURL:   adminURL,
		nodeID:     nodeID,
		token:      token,
		certLoader: certLoader,
	}
}

func (b *Bootstrapper) Bootstrap(ctx context.Context) error {
	if b.certLoader == nil {
		return fmt.Errorf("cert loader is required")
	}

	if b.adminURL == "" {
		return fmt.Errorf("admin URL is required")
	}

	if b.token == "" {
		return fmt.Errorf("bootstrap token is required - check your configuration")
	}

	if b.nodeID == "" {
		return fmt.Errorf("node ID is required")
	}

	url := fmt.Sprintf("%s/api/nodes/%s/bootstrap", b.adminURL, b.nodeID)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
		}

		req := BootstrapRequest{Token: b.token}
		data, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal bootstrap request: %w", err)
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("network error (attempt %d/3): %w", attempt+1, err)
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("invalid bootstrap token - please check your configuration")
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("node not found - please verify node ID")
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("bootstrap failed with status %d: %s", resp.StatusCode, string(body))
			continue
		}

		var bootstrapResp BootstrapResponse
		if err := json.NewDecoder(resp.Body).Decode(&bootstrapResp); err != nil {
			return fmt.Errorf("failed to decode bootstrap response: %w", err)
		}

		certPEM := []byte(bootstrapResp.Cert)
		keyPEM := []byte(bootstrapResp.Key)

		if err := cert.Save(certPEM, keyPEM, b.certLoader.CertFile, b.certLoader.KeyFile); err != nil {
			return fmt.Errorf("failed to save certificate: %w", err)
		}

		if bootstrapResp.CACert != "" && b.certLoader.CACert == nil {
			caCertPath := b.certLoader.CertFile[:len(b.certLoader.CertFile)-4] + "ca.pem"
			if err := os.WriteFile(caCertPath, []byte(bootstrapResp.CACert), 0600); err != nil {
				return fmt.Errorf("failed to save CA certificate: %w", err)
			}
		}

		return nil
	}

	return lastErr
}

func (b *Bootstrapper) IsBootstrapped() bool {
	if b.certLoader == nil {
		return false
	}

	if b.certLoader.CertFile == "" || b.certLoader.KeyFile == "" {
		return false
	}

	_, err := os.Stat(b.certLoader.CertFile)
	if err != nil {
		return false
	}

	_, err = os.Stat(b.certLoader.KeyFile)
	if err != nil {
		return false
	}

	existingCert, err := cert.LoadFromFiles(b.certLoader.CertFile, b.certLoader.KeyFile, "")
	if err != nil {
		return false
	}

	if existingCert.IsExpired() {
		return false
	}

	return existingCert.Verify() == nil
}