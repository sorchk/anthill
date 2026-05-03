package handler

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type NodeConnectionInfo struct {
	NodeID        string
	Conn          net.Conn
	Protocol      string
	LastHeartbeat time.Time
	Mode          string
}

type ConnManagerInterface interface {
	GetConnection(nodeID string) (*NodeConnectionInfo, bool)
	ConnectActiveNode(nodeID, addr string, port int, protocol string, cert *tls.Certificate, caCert *x509.Certificate) error
}

type NodeHandler struct {
	DB       *gorm.DB
	connMgr  ConnManagerInterface
	caCert   *tls.Certificate
	caCertX509 *x509.Certificate
}

func NewNodeHandler(db *gorm.DB, connMgr ConnManagerInterface, caCert *tls.Certificate, caCertX509 *x509.Certificate) *NodeHandler {
	return &NodeHandler{DB: db, connMgr: connMgr, caCert: caCert, caCertX509: caCertX509}
}

func (h *NodeHandler) getUserID(c *gin.Context) int64 {
	uid, _ := strconv.ParseInt(c.GetString("user_id"), 10, 64)
	return uid
}

func (h *NodeHandler) getUserRole(c *gin.Context) string {
	return c.GetString("role")
}

func (h *NodeHandler) List(c *gin.Context) {
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var nodes []model.Node
	err := h.DB.Order("name").Find(&nodes).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var filtered []model.Node
	for i := range nodes {
		if userRole == "admin" || nodes[i].CanView(userID) {
			if nodes[i].LastSeen != nil {
				nodes[i].Status = computeStatus(*nodes[i].LastSeen)
			}
			filtered = append(filtered, nodes[i])
		}
	}

	c.JSON(http.StatusOK, filtered)
}

func (h *NodeHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	err := h.DB.First(&n, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if userRole != "admin" && !n.CanView(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	if n.LastSeen != nil {
		n.Status = computeStatus(*n.LastSeen)
	}

	c.JSON(http.StatusOK, n)
}

func (h *NodeHandler) Create(c *gin.Context) {
	userID := h.getUserID(c)

	var req struct {
		Name        string  `json:"name" binding:"required"`
		Host        string  `json:"host" binding:"required"`
		Port        int     `json:"port"`
		SSHHost     string  `json:"ssh_host"`
		SSHPort     int     `json:"ssh_port"`
		SSHUsername string  `json:"ssh_username"`
		TLSCertPath string  `json:"tls_cert_path"`
		TLSCertCN   string  `json:"tls_cert_cn"`
		NodeGroup   string  `json:"node_group"`
		IsPrivate   bool    `json:"is_private"`
		VisibleTo   []int64 `json:"visible_to"`
		HiddenFrom  []int64 `json:"hidden_from"`
		ConnectMode string  `json:"connect_mode"`
		NodePort    int     `json:"node_port"`
		NodeHost    string  `json:"node_host"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	port := req.Port
	if port == 0 {
		port = 18888
	}
	nodePort := req.NodePort
	if nodePort == 0 {
		nodePort = 18888
	}
	connectMode := req.ConnectMode
	if connectMode == "" {
		connectMode = model.ConnectModePassiveTLS
	}
	if !model.ValidConnectMode(connectMode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid connect_mode. Must be one of: active_tls, active_wss, passive_tls, passive_wss, auto"})
		return
	}

	visibleToJSON, _ := json.Marshal(req.VisibleTo)
	hiddenFromJSON, _ := json.Marshal(req.HiddenFrom)

	node := &model.Node{
		Name:         req.Name,
		Host:         req.Host,
		Port:         port,
		SSHHost:      req.SSHHost,
		SSHPort:      req.SSHPort,
		SSHUsername:  req.SSHUsername,
		TLSCertPath:  req.TLSCertPath,
		TLSCertCN:    req.TLSCertCN,
		Status:       "unknown",
		NodeGroup:    req.NodeGroup,
		IsPrivate:    req.IsPrivate,
		OwnerID:      userID,
		VisibleTo:    string(visibleToJSON),
		HiddenFrom:   string(hiddenFromJSON),
		ConnectMode:  connectMode,
		NodePort:     nodePort,
		NodeHost:     req.NodeHost,
		BootstrapToken: uuid.New().String(),
	}

	if err := h.DB.Create(node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": node.ID})
}

func (h *NodeHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	err := h.DB.First(&n, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !n.CanEdit(userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner or admin can update"})
		return
	}

	var req struct {
		Name        string   `json:"name"`
		Host        string   `json:"host"`
		Port        int      `json:"port"`
		NodeGroup   string   `json:"node_group"`
		IsPrivate   *bool    `json:"is_private"`
		VisibleTo   []int64  `json:"visible_to"`
		HiddenFrom  []int64  `json:"hidden_from"`
		ConnectMode string   `json:"connect_mode"`
		NodePort    int      `json:"node_port"`
		NodeHost    string   `json:"node_host"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Host != "" {
		updates["host"] = req.Host
	}
	if req.Port != 0 {
		updates["port"] = req.Port
	}
	if req.NodeGroup != "" {
		updates["node_group"] = req.NodeGroup
	}
	if req.IsPrivate != nil && n.IsOwner(userID) {
		updates["is_private"] = *req.IsPrivate
	}
	if req.VisibleTo != nil || req.HiddenFrom != nil {
		visibleToJSON, _ := json.Marshal(req.VisibleTo)
		hiddenFromJSON, _ := json.Marshal(req.HiddenFrom)
		updates["visible_to"] = string(visibleToJSON)
		updates["hidden_from"] = string(hiddenFromJSON)
	}
	if req.ConnectMode != "" {
		if !model.ValidConnectMode(req.ConnectMode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid connect_mode"})
			return
		}
		updates["connect_mode"] = req.ConnectMode
	}
	if req.NodePort != 0 {
		updates["node_port"] = req.NodePort
	}
	if req.NodeHost != "" {
		updates["node_host"] = req.NodeHost
	}

	updates["updated_at"] = time.Now()

	if len(updates) > 0 {
		h.DB.Model(&n).Updates(updates)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated"})
}

func (h *NodeHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	err := h.DB.First(&n, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !n.CanEdit(userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner or admin can delete"})
		return
	}

	h.DB.Delete(&n)

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

func (h *NodeHandler) Connect(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	err := h.DB.First(&n, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if userRole != "admin" && !n.CanView(userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	nodeIDStr := strconv.FormatInt(id, 10)

	if h.connMgr != nil {
		if existingConn, ok := h.connMgr.GetConnection(nodeIDStr); ok && existingConn != nil {
			now := time.Now()
			h.DB.Model(&n).Update("last_seen", now)
			c.JSON(http.StatusOK, gin.H{
				"status":  "online",
				"message": "Node is connected",
				"node_id": id,
			})
			return
		}

		if n.ConnectMode == model.ConnectModeActiveTLS || n.ConnectMode == model.ConnectModeActiveWSS {
			protocol := "tls"
			if n.ConnectMode == model.ConnectModeActiveWSS {
				protocol = "wss"
			}
			addr := n.Host
			port := n.Port
			if n.NodePort > 0 {
				port = n.NodePort
			}
			if n.NodeHost != "" {
				addr = n.NodeHost
			}

			err := h.connMgr.ConnectActiveNode(nodeIDStr, addr, port, protocol, nil, h.caCertX509)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"status":  "offline",
					"message": "Failed to connect: " + err.Error(),
				})
				return
			}

			now := time.Now()
			h.DB.Model(&n).Updates(map[string]interface{}{
				"last_seen":      now,
				"last_conn_mode": n.ConnectMode,
				"updated_at":     now,
			})
			c.JSON(http.StatusOK, gin.H{
				"status":  "online",
				"message": "Node connected successfully",
				"node_id": id,
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "offline",
			"message": "Node is not connected. Please ensure the runtime is installed and running on the node.",
		})
		return
	}

	now := time.Now()
	h.DB.Model(&n).Update("last_seen", now)

	c.JSON(http.StatusOK, gin.H{
		"status":  "online",
		"message": "Node is connected",
		"node_id": id,
	})
}

func (h *NodeHandler) ResetToken(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	err := h.DB.First(&n, id).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !n.CanEdit(userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner or admin can reset token"})
		return
	}

	newToken := uuid.New().String()
	h.DB.Model(&n).Update("bootstrap_token", newToken)

	c.JSON(http.StatusOK, gin.H{
		"message":        "Token reset successfully",
		"bootstrap_token": newToken,
	})
}

func computeStatus(lastSeen time.Time) string {
	if time.Since(lastSeen) < 2*time.Minute {
		return "online"
	}
	return "offline"
}