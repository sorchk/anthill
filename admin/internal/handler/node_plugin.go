package handler

import (
    "database/sql"
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"

    "anthill/admin/internal/model"
)

type NodePluginHandler struct {
    DB *sql.DB
}

func NewNodePluginHandler(db *sql.DB) *NodePluginHandler {
    return &NodePluginHandler{DB: db}
}

func (h *NodePluginHandler) ListForNode(c *gin.Context) {
    nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

    rows, err := h.DB.Query(`
        SELECT np.id, np.node_id, np.plugin_name, np.version, p.plugin_type, np.status, np.enabled, np.installed_at
        FROM node_plugins np
        JOIN plugins p ON np.plugin_name = p.name AND np.version = p.version
        WHERE np.node_id = ?
    `, nodeID)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    defer rows.Close()

    var plugins []model.NodePlugin
    for rows.Next() {
        var p model.NodePlugin
        rows.Scan(&p.ID, &p.NodeID, &p.PluginName, &p.Version, &p.PluginType, &p.Status, &p.Enabled, &p.InstalledAt)
        plugins = append(plugins, p)
    }

    c.JSON(http.StatusOK, plugins)
}

func (h *NodePluginHandler) Install(c *gin.Context) {
    nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

    var req model.NodePluginInstall
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var pluginID int64
    var pluginType string
    err := h.DB.QueryRow(`
        SELECT id, plugin_type FROM plugins WHERE name = ? AND version = ?
    `, req.PluginName, req.Version).Scan(&pluginID, &pluginType)

    if err == sql.ErrNoRows {
        c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found in repository"})
        return
    }

    result, err := h.DB.Exec(`
        INSERT INTO node_plugins (node_id, plugin_name, version, plugin_type, status, enabled, installed_at)
        VALUES (?, ?, ?, ?, 'installing', true, CURRENT_TIMESTAMP)
        ON CONFLICT(node_id, plugin_name) DO UPDATE SET version = ?, plugin_type = ?, status = 'installing'
    `, nodeID, req.PluginName, req.Version, pluginType, req.Version, pluginType)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    taskID, _ := result.LastInsertId()

    c.JSON(http.StatusAccepted, gin.H{
        "task_id": taskID,
        "message": "Plugin installation started",
        "node_id": nodeID,
        "plugin": req.PluginName,
        "version": req.Version,
    })
}

func (h *NodePluginHandler) Uninstall(c *gin.Context) {
    nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    pluginName := c.Param("plugin")

    result, err := h.DB.Exec(`
        DELETE FROM node_plugins WHERE node_id = ? AND plugin_name = ?
    `, nodeID, pluginName)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    rows, _ := result.RowsAffected()
    if rows == 0 {
        c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not installed on this node"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Plugin uninstalled"})
}

func (h *NodePluginHandler) Enable(c *gin.Context) {
    nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    pluginName := c.Param("plugin")

    _, err := h.DB.Exec(`
        UPDATE node_plugins SET enabled = true WHERE node_id = ? AND plugin_name = ?
    `, nodeID, pluginName)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Plugin enabled"})
}

func (h *NodePluginHandler) Disable(c *gin.Context) {
    nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
    pluginName := c.Param("plugin")

    _, err := h.DB.Exec(`
        UPDATE node_plugins SET enabled = false WHERE node_id = ? AND plugin_name = ?
    `, nodeID, pluginName)

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Plugin disabled"})
}