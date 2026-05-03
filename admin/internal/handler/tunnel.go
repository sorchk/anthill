package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type TunnelHandler struct {
	DB *gorm.DB
}

func NewTunnelHandler(db *gorm.DB) *TunnelHandler {
	return &TunnelHandler{DB: db}
}

func (h *TunnelHandler) List(c *gin.Context) {
	var tunnels []model.Tunnel
	err := h.DB.Order("created_at DESC").Find(&tunnels).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tunnels)
}

func (h *TunnelHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var tunnel model.Tunnel
	err := h.DB.First(&tunnel, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tunnel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tunnel)
}

func (h *TunnelHandler) Create(c *gin.Context) {
	var req struct {
		Name            string `json:"name" binding:"required"`
		Type            string `json:"type" binding:"required"`
		TransportMode   string `json:"transport_mode"`
		LocalAddr       string `json:"local_addr"`
		RemoteAddr      string `json:"remote_addr"`
		ObfuscationMode string `json:"obfuscation_mode"`
		ObfuscationCfg  string `json:"obfuscation_config"`
		E2EEnabled      bool   `json:"e2e_enabled"`
		TLSFingerprint  bool   `json:"tls_fingerprint"`
		TLSProfile      string `json:"tls_profile"`
		NodeID          string `json:"node_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tunnelType := req.Type
	if tunnelType != "port_forward" && tunnelType != "socks5" && tunnelType != "http" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of: port_forward, socks5, http"})
		return
	}

	transportMode := req.TransportMode
	if transportMode == "" {
		transportMode = "auto"
	}
	if transportMode != "direct" && transportMode != "relay" && transportMode != "auto" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "transport_mode must be one of: direct, relay, auto"})
		return
	}

	obfuscationMode := req.ObfuscationMode
	if obfuscationMode == "" {
		obfuscationMode = "none"
	}

	tunnelID := "tun_" + uuid.New().String()[:12]

	tunnel := &model.Tunnel{
		TunnelID:        tunnelID,
		Name:            req.Name,
		Type:            tunnelType,
		TransportMode:   transportMode,
		LocalAddr:       req.LocalAddr,
		RemoteAddr:      req.RemoteAddr,
		ObfuscationMode: obfuscationMode,
		ObfuscationCfg:  req.ObfuscationCfg,
		E2EEnabled:      req.E2EEnabled,
		TLSFingerprint:  req.TLSFingerprint,
		TLSProfile:      req.TLSProfile,
		Status:          "active",
		NodeID:          req.NodeID,
	}

	if err := h.DB.Create(tunnel).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tunnel)
}

func (h *TunnelHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var tunnel model.Tunnel
	err := h.DB.First(&tunnel, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tunnel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Name            string `json:"name"`
		Type            string `json:"type"`
		TransportMode   string `json:"transport_mode"`
		LocalAddr       string `json:"local_addr"`
		RemoteAddr      string `json:"remote_addr"`
		ObfuscationMode string `json:"obfuscation_mode"`
		ObfuscationCfg  string `json:"obfuscation_config"`
		E2EEnabled      *bool  `json:"e2e_enabled"`
		TLSFingerprint  *bool   `json:"tls_fingerprint"`
		TLSProfile      string `json:"tls_profile"`
		Status          string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		if req.Type != "port_forward" && req.Type != "socks5" && req.Type != "http" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of: port_forward, socks5, http"})
			return
		}
		updates["type"] = req.Type
	}
	if req.TransportMode != "" {
		if req.TransportMode != "direct" && req.TransportMode != "relay" && req.TransportMode != "auto" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "transport_mode must be one of: direct, relay, auto"})
			return
		}
		updates["transport_mode"] = req.TransportMode
	}
	if req.LocalAddr != "" {
		updates["local_addr"] = req.LocalAddr
	}
	if req.RemoteAddr != "" {
		updates["remote_addr"] = req.RemoteAddr
	}
	if req.ObfuscationMode != "" {
		updates["obfuscation_mode"] = req.ObfuscationMode
	}
	if req.ObfuscationCfg != "" {
		updates["obfuscation_config"] = req.ObfuscationCfg
	}
	if req.E2EEnabled != nil {
		updates["e2e_enabled"] = *req.E2EEnabled
	}
	if req.TLSFingerprint != nil {
		updates["tls_fingerprint"] = *req.TLSFingerprint
	}
	if req.TLSProfile != "" {
		updates["tls_profile"] = req.TLSProfile
	}
	if req.Status != "" {
		if req.Status != "active" && req.Status != "paused" && req.Status != "stopped" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status must be one of: active, paused, stopped"})
			return
		}
		updates["status"] = req.Status
	}

	if len(updates) > 0 {
		h.DB.Model(&tunnel).Updates(updates)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated"})
}

func (h *TunnelHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var tunnel model.Tunnel
	err := h.DB.First(&tunnel, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tunnel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.DB.Delete(&tunnel)

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

func (h *TunnelHandler) Toggle(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var tunnel model.Tunnel
	err := h.DB.First(&tunnel, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tunnel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newStatus := "active"
	if tunnel.Status == "active" {
		newStatus = "paused"
	}

	h.DB.Model(&tunnel).Update("status", newStatus)

	c.JSON(http.StatusOK, gin.H{"status": newStatus})
}

func (h *TunnelHandler) Stats(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var tunnel model.Tunnel
	err := h.DB.First(&tunnel, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tunnel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tunnel_id": tunnel.TunnelID,
		"bytes_in":  tunnel.BytesIn,
		"bytes_out": tunnel.BytesOut,
		"status":    tunnel.Status,
	})
}