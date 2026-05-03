package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"anthill/admin/internal/model"
)

type NodeHandler struct {
	DB *sql.DB
}

func NewNodeHandler(db *sql.DB) *NodeHandler {
	return &NodeHandler{DB: db}
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

	rows, err := h.DB.Query(`
		SELECT id, name, host, port, ssh_host, ssh_port, ssh_username,
			   tls_cert_path, tls_cert_cn, status, last_seen, node_group,
			   is_private, owner_id, visible_to_users, hidden_from_users,
			   created_at, updated_at,
			   connect_mode, node_port, node_host, bootstrap_token
		FROM nodes ORDER BY name
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var nodes []model.Node
	for rows.Next() {
		var n model.Node
		var sshHost, sshUser, tlsCert, tlsCN, nodeGroup, visibleTo, hiddenFrom sql.NullString
		var sshPort sql.NullInt64
		var lastSeen sql.NullTime
		var isPrivate, ownerID sql.NullInt64
		var connectMode, nodeHost, bootstrapToken sql.NullString
		var nodePort sql.NullInt64

		err := rows.Scan(&n.ID, &n.Name, &n.Host, &n.Port, &sshHost, &sshPort,
			&sshUser, &tlsCert, &tlsCN, &n.Status, &lastSeen, &nodeGroup,
			&isPrivate, &ownerID, &visibleTo, &hiddenFrom,
			&n.CreatedAt, &n.UpdatedAt,
			&connectMode, &nodePort, &nodeHost, &bootstrapToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if sshHost.Valid {
			n.SSHHost = sshHost.String
		}
		if sshPort.Valid {
			n.SSHPort = int(sshPort.Int64)
		}
		if sshUser.Valid {
			n.SSHUsername = sshUser.String
		}
		if tlsCert.Valid {
			n.TLSCertPath = tlsCert.String
		}
		if tlsCN.Valid {
			n.TLSCertCN = tlsCN.String
		}
		if lastSeen.Valid {
			n.LastSeen = lastSeen.Time
			n.Status = computeStatus(n.LastSeen)
		}
		if nodeGroup.Valid {
			n.NodeGroup = nodeGroup.String
		}
		if isPrivate.Valid {
			n.IsPrivate = isPrivate.Int64 == 1
		}
		if ownerID.Valid {
			n.OwnerID = ownerID.Int64
		}
		if visibleTo.Valid {
			n.VisibleTo, _ = model.ParseVisibleTo(visibleTo.String)
		}
		if hiddenFrom.Valid {
			n.HiddenFrom, _ = model.ParseHiddenFrom(hiddenFrom.String)
		}
		if connectMode.Valid {
			n.ConnectMode = connectMode.String
		}
		if nodePort.Valid {
			n.NodePort = int(nodePort.Int64)
		}
		if nodeHost.Valid {
			n.NodeHost = nodeHost.String
		}
		if bootstrapToken.Valid {
			n.BootstrapToken = bootstrapToken.String
		}

		if userRole == "admin" || n.CanView(userID) {
			nodes = append(nodes, n)
		}
	}

	c.JSON(http.StatusOK, nodes)
}

func (h *NodeHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	var sshHost, sshUser, tlsCert, tlsCN, nodeGroup, visibleTo, hiddenFrom sql.NullString
	var sshPort sql.NullInt64
	var lastSeen sql.NullTime
	var isPrivate, ownerID sql.NullInt64
	var connectMode, nodeHost, bootstrapToken sql.NullString
	var nodePort sql.NullInt64

	err := h.DB.QueryRow(`
		SELECT id, name, host, port, ssh_host, ssh_port, ssh_username,
			   tls_cert_path, tls_cert_cn, status, last_seen, node_group,
			   is_private, owner_id, visible_to_users, hidden_from_users,
			   created_at, updated_at,
			   connect_mode, node_port, node_host, bootstrap_token
		FROM nodes WHERE id = ?
	`, id).Scan(&n.ID, &n.Name, &n.Host, &n.Port, &sshHost, &sshPort,
		&sshUser, &tlsCert, &tlsCN, &n.Status, &lastSeen, &nodeGroup,
		&isPrivate, &ownerID, &visibleTo, &hiddenFrom, &n.CreatedAt, &n.UpdatedAt,
		&connectMode, &nodePort, &nodeHost, &bootstrapToken)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if sshHost.Valid {
		n.SSHHost = sshHost.String
	}
	if sshPort.Valid {
		n.SSHPort = int(sshPort.Int64)
	}
	if sshUser.Valid {
		n.SSHUsername = sshUser.String
	}
	if tlsCert.Valid {
		n.TLSCertPath = tlsCert.String
	}
	if tlsCN.Valid {
		n.TLSCertCN = tlsCN.String
	}
	if lastSeen.Valid {
		n.LastSeen = lastSeen.Time
		n.Status = computeStatus(n.LastSeen)
	}
	if nodeGroup.Valid {
		n.NodeGroup = nodeGroup.String
	}
	if isPrivate.Valid {
		n.IsPrivate = isPrivate.Int64 == 1
	}
	if ownerID.Valid {
		n.OwnerID = ownerID.Int64
	}
	if visibleTo.Valid {
		n.VisibleTo, _ = model.ParseVisibleTo(visibleTo.String)
	}
	if hiddenFrom.Valid {
		n.HiddenFrom, _ = model.ParseHiddenFrom(hiddenFrom.String)
	}
	if connectMode.Valid {
		n.ConnectMode = connectMode.String
	}
	if nodePort.Valid {
		n.NodePort = int(nodePort.Int64)
	}
	if nodeHost.Valid {
		n.NodeHost = nodeHost.String
	}
	if bootstrapToken.Valid {
		n.BootstrapToken = bootstrapToken.String
	}

	if userRole == "admin" || n.CanView(userID) {
		c.JSON(http.StatusOK, n)
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
	}
}

func (h *NodeHandler) Create(c *gin.Context) {
	userID := h.getUserID(c)

	var req struct {
		Name        string   `json:"name" binding:"required"`
		Host        string   `json:"host" binding:"required"`
		Port        int      `json:"port"`
		SSHHost     string   `json:"ssh_host"`
		SSHPort     int      `json:"ssh_port"`
		SSHUsername string   `json:"ssh_username"`
		TLSCertPath string   `json:"tls_cert_path"`
		TLSCertCN   string   `json:"tls_cert_cn"`
		NodeGroup   string   `json:"node_group"`
		IsPrivate   bool     `json:"is_private"`
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

	if req.Port == 0 {
		req.Port = 18888
	}
	if req.NodePort == 0 {
		req.NodePort = 18888
	}
	if req.ConnectMode == "" {
		req.ConnectMode = "passive"
	}

	bootstrapToken := uuid.New().String()
	visibleToJSON, _ := json.Marshal(req.VisibleTo)
	hiddenFromJSON, _ := json.Marshal(req.HiddenFrom)

	result, err := h.DB.Exec(`
		INSERT INTO nodes (name, host, port, ssh_host, ssh_port, ssh_username, tls_cert_path, tls_cert_cn, status, node_group, is_private, owner_id, visible_to_users, hidden_from_users, connect_mode, node_port, node_host, bootstrap_token)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'unknown', ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.Name, req.Host, req.Port, req.SSHHost, req.SSHPort, req.SSHUsername, req.TLSCertPath, req.TLSCertCN, req.NodeGroup, boolToInt(req.IsPrivate), userID, string(visibleToJSON), string(hiddenFromJSON), req.ConnectMode, req.NodePort, req.NodeHost, bootstrapToken)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *NodeHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	var ownerID sql.NullInt64
	err := h.DB.QueryRow("SELECT owner_id FROM nodes WHERE id = ?", id).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if !n.CanEdit(userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner or admin can update"})
		return
	}

	var req struct {
		Name       string   `json:"name"`
		Host       string   `json:"host"`
		Port       int      `json:"port"`
		NodeGroup  string   `json:"node_group"`
		IsPrivate  *bool    `json:"is_private"`
		VisibleTo  []int64  `json:"visible_to"`
		HiddenFrom []int64  `json:"hidden_from"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IsPrivate != nil && n.IsOwner(userID) {
		_, err = h.DB.Exec("UPDATE nodes SET is_private = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", boolToInt(*req.IsPrivate), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if req.VisibleTo != nil || req.HiddenFrom != nil {
		visibleToJSON, _ := json.Marshal(req.VisibleTo)
		hiddenFromJSON, _ := json.Marshal(req.HiddenFrom)
		_, err = h.DB.Exec("UPDATE nodes SET visible_to_users = ?, hidden_from_users = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", string(visibleToJSON), string(hiddenFromJSON), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	_, err = h.DB.Exec(`
		UPDATE nodes SET name = ?, host = ?, port = ?, node_group = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, req.Name, req.Host, req.Port, req.NodeGroup, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Updated"})
}

func (h *NodeHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	var ownerID sql.NullInt64
	err := h.DB.QueryRow("SELECT owner_id FROM nodes WHERE id = ?", id).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if !n.CanEdit(userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only owner or admin can delete"})
		return
	}

	_, err = h.DB.Exec("DELETE FROM nodes WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

func (h *NodeHandler) Connect(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := h.getUserID(c)
	userRole := h.getUserRole(c)

	var n model.Node
	var ownerID sql.NullInt64
	err := h.DB.QueryRow("SELECT owner_id FROM nodes WHERE id = ?", id).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}
	if !n.CanView(userID) && userRole != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.DB.Exec("UPDATE nodes SET last_seen = CURRENT_TIMESTAMP WHERE id = ?", id)

	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"message": "Connection check recorded",
		"node_id": id,
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func computeStatus(lastSeen time.Time) string {
	if time.Since(lastSeen) < 2*time.Minute {
		return "online"
	}
	return "offline"
}