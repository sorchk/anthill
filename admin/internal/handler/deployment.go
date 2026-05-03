package handler

import (
    "database/sql"
    "net/http"
    "strconv"
    "time"

    "github.com/gin-gonic/gin"

    "anthill/admin/internal/model"
)

type DeploymentHandler struct {
    DB *sql.DB
}

func NewDeploymentHandler(db *sql.DB) *DeploymentHandler {
    return &DeploymentHandler{DB: db}
}

func (h *DeploymentHandler) List(c *gin.Context) {
    nodeID := c.Query("node_id")
    pluginID := c.Query("plugin_id")

    query := `
        SELECT d.id, d.node_id, d.plugin_id, d.version, d.status, d.result,
               d.deployed_by, d.created_at, d.updated_at,
               n.name as node_name, p.name as plugin_name
        FROM deployments d
        LEFT JOIN nodes n ON d.node_id = n.id
        LEFT JOIN plugins p ON d.plugin_id = p.id
        WHERE 1=1
    `
    args := []interface{}{}

    if nodeID != "" {
        query += " AND d.node_id = ?"
        args = append(args, nodeID)
    }
    if pluginID != "" {
        query += " AND d.plugin_id = ?"
        args = append(args, pluginID)
    }

    query += " ORDER BY d.created_at DESC LIMIT 100"

    rows, err := h.DB.Query(query, args...)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    type DeploymentWithNames struct {
        model.Deployment
        NodeName   string `json:"node_name"`
        PluginName string `json:"plugin_name"`
    }

    var deployments []DeploymentWithNames
    for rows.Next() {
        var d DeploymentWithNames
        var result sql.NullString
        var nodeName, pluginName sql.NullString

        err := rows.Scan(&d.ID, &d.NodeID, &d.PluginID, &d.Version, &d.Status,
            &result, &d.DeployedBy, &d.CreatedAt, &d.UpdatedAt, &nodeName, &pluginName)
        if err != nil {
            continue
        }

        if result.Valid {
            d.Result = result.String
        }
        if nodeName.Valid {
            d.NodeName = nodeName.String
        }
        if pluginName.Valid {
            d.PluginName = pluginName.String
        }

        deployments = append(deployments, d)
    }

    c.JSON(http.StatusOK, deployments)
}

func (h *DeploymentHandler) Create(c *gin.Context) {
    var req model.DeploymentRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    userID, _ := c.Get("user_id")

    var pluginVersion string
    err := h.DB.QueryRow("SELECT version FROM plugins WHERE id = ?", req.PluginID).Scan(&pluginVersion)
    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var deployments []int64
    for _, nodeID := range req.NodeIDs {
result, _ := h.DB.Exec(`
            INSERT INTO deployments (node_id, plugin_id, version, status, deployed_by)
            VALUES (?, ?, ?, 'pending', ?)
        `, nodeID, req.PluginID, pluginVersion, userID)

        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create deployment: " + err.Error()})
            return
        }

        id, _ := result.LastInsertId()
        deployments = append(deployments, id)

        go h.processDeployment(id, nodeID, req.PluginID)
    }

    c.JSON(http.StatusAccepted, gin.H{
        "message":      "Deployment initiated",
        "deployments": deployments,
    })
}

func (h *DeploymentHandler) processDeployment(deploymentID, nodeID, pluginID int64) {
    time.Sleep(2 * time.Second)

    _, err := h.DB.Exec(`
        UPDATE deployments SET status = 'deployed', updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `, deploymentID)

    if err != nil {
        h.DB.Exec(`
            UPDATE deployments SET status = 'failed', result = ?, updated_at = CURRENT_TIMESTAMP
            WHERE id = ?
        `, err.Error(), deploymentID)
    }
}

func (h *DeploymentHandler) Get(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

    var d model.Deployment
    var result sql.NullString

    err := h.DB.QueryRow(`
        SELECT id, node_id, plugin_id, version, status, result, deployed_by, created_at, updated_at
        FROM deployments WHERE id = ?
    `, id).Scan(&d.ID, &d.NodeID, &d.PluginID, &d.Version, &d.Status, &result, &d.DeployedBy, &d.CreatedAt, &d.UpdatedAt)

    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
        return
    }
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    if result.Valid {
        d.Result = result.String
    }

    c.JSON(http.StatusOK, d)
}

func (h *DeploymentHandler) Cancel(c *gin.Context) {
    id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

    result, _ := h.DB.Exec(`
        UPDATE deployments SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP
        WHERE id = ? AND status = 'pending'
    `, id)

    affected, _ := result.RowsAffected()
    if affected == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel - deployment not pending or not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Cancelled"})
}