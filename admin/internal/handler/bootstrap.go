package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type BootstrapHandler struct {
	DB *gorm.DB
	ca *CertCA
}

func NewBootstrapHandler(db *gorm.DB, ca *CertCA) *BootstrapHandler {
	return &BootstrapHandler{DB: db, ca: ca}
}

func (h *BootstrapHandler) Bootstrap(c *gin.Context) {
	nodeID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
		return
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
		return
	}

	bootstrapToken := strings.TrimPrefix(authHeader, "Bearer ")

	var node model.Node
	err = h.DB.Where("id = ?", nodeID).First(&node).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if node.BootstrapToken != bootstrapToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid bootstrap token"})
		return
	}

	expires := time.Now().Add(365 * 24 * time.Hour)

	certPEM, keyPEM, serial, err := h.ca.SignNodeCert(strconv.FormatInt(nodeID, 10), expires)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign certificate"})
		return
	}

	err = h.DB.Model(&model.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
		"cert_serial":  serial,
		"cert_expires": expires,
		"updated_at":   time.Now(),
	}).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store certificate info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cert":    string(certPEM),
		"key":     string(keyPEM),
		"expires": expires.Format(time.RFC3339),
	})
}