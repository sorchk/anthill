package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type NodePluginHandler struct {
	DB *gorm.DB
}

func NewNodePluginHandler(db *gorm.DB) *NodePluginHandler {
	return &NodePluginHandler{DB: db}
}

func (h *NodePluginHandler) ListForNode(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	plugins, err := model.ListNodePlugins(h.DB, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	var plugin model.Plugin
	err := h.DB.Where("name = ? AND version = ?", req.PluginName, req.Version).First(&plugin).Error
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found in repository"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	nodePlugin := &model.NodePlugin{
		NodeID:     nodeID,
		PluginName: req.PluginName,
		Version:    req.Version,
		PluginType: plugin.PluginType,
		Status:     "installing",
		Enabled:    true,
	}

	if err := model.CreateOrUpdateNodePlugin(h.DB, nodePlugin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "Plugin installation started",
		"node_id":  nodeID,
		"plugin":   req.PluginName,
		"version":  req.Version,
	})
}

func (h *NodePluginHandler) Uninstall(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	pluginName := c.Param("plugin")

	if err := model.DeleteNodePlugin(h.DB, nodeID, pluginName); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not installed on this node"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plugin uninstalled"})
}

func (h *NodePluginHandler) Enable(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	pluginName := c.Param("plugin")

	if err := model.SetNodePluginEnabled(h.DB, nodeID, pluginName, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plugin enabled"})
}

func (h *NodePluginHandler) Disable(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	pluginName := c.Param("plugin")

	if err := model.SetNodePluginEnabled(h.DB, nodeID, pluginName, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Plugin disabled"})
}