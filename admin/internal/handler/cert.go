package handler

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type CertHandler struct {
	DB *gorm.DB
	ca *CertCA
}

func NewCertHandler(db *gorm.DB, ca *CertCA) *CertHandler {
	return &CertHandler{DB: db, ca: ca}
}

func (h *CertHandler) GetCACert() *tls.Certificate {
	if h.ca == nil {
		return nil
	}
	return h.ca.GetCACertTLS()
}

func (h *CertHandler) GetCACertX509() *x509.Certificate {
	if h.ca == nil {
		return nil
	}
	return h.ca.GetCACertX509()
}

func (h *CertHandler) RenewCert(c *gin.Context) {
	nodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	var node model.Node
	if err := h.DB.First(&node, nodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	var req struct {
		Days int `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Days = 365
	}
	if req.Days <= 0 {
		req.Days = 365
	}

	expires := time.Now().Add(time.Duration(req.Days) * 24 * time.Hour)

	certPEM, keyPEM, serial, err := h.ca.RenewNodeCert(strconv.FormatInt(nodeID, 10), expires)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to renew certificate"})
		return
	}

	if err := h.DB.Model(&node).Updates(map[string]interface{}{
		"cert_serial":  serial,
		"cert_expires": expires,
		"updated_at":   time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update node certificate info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cert":    string(certPEM),
		"key":     string(keyPEM),
		"ca_cert": string(h.ca.GetCACert()),
		"expires": expires.Format(time.RFC3339),
		"serial":  serial,
	})
}

func (h *CertHandler) RevokeCert(c *gin.Context) {
	nodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	var node model.Node
	if err := h.DB.First(&node, nodeID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	if node.CertSerial == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Node has no certificate to revoke"})
		return
	}

	if err := h.ca.RevokeCert(node.CertSerial); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke certificate"})
		return
	}

	if err := h.DB.Model(&node).Updates(map[string]interface{}{
		"cert_serial":  "",
		"cert_expires": nil,
		"updated_at":   time.Now(),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update node certificate info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Certificate revoked successfully"})
}

func (h *CertHandler) GetCRL(c *gin.Context) {
	crl, err := h.ca.GenerateCRL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate CRL"})
		return
	}

	c.Data(http.StatusOK, "application/x-x509-ca-cert", crl)
}