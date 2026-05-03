package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type BootstrapHandler struct {
	DB *sql.DB
	ca *CertCA
}

func NewBootstrapHandler(db *sql.DB, ca *CertCA) *BootstrapHandler {
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

	var storedToken string
	err = h.DB.QueryRow("SELECT bootstrap_token FROM nodes WHERE id = ?", nodeID).Scan(&storedToken)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if storedToken != bootstrapToken {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid bootstrap token"})
		return
	}

	expires := time.Now().Add(365 * 24 * time.Hour)

	certPEM, keyPEM, serial, err := h.ca.SignNodeCert(strconv.FormatInt(nodeID, 10), expires)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign certificate"})
		return
	}

	_, err = h.DB.Exec(`
		UPDATE nodes
		SET cert_serial = ?, cert_expires = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, serial, expires, nodeID)
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