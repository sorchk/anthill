package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ServicePluginHandler struct {
	DB *gorm.DB
}

func NewServicePluginHandler(db *gorm.DB) *ServicePluginHandler {
	return &ServicePluginHandler{DB: db}
}

type ServiceStatus struct {
	Plugin   string `json:"plugin"`
	Status   string `json:"status"`
	PID      int    `json:"pid,omitempty"`
	Uptime   int64  `json:"uptime_seconds"`
	MemoryMB int    `json:"memory_mb,omitempty"`
}

func (h *ServicePluginHandler) Start(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	plugin := c.Param("plugin")

	c.JSON(http.StatusOK, gin.H{
		"node_id": nodeID,
		"plugin":  plugin,
		"action":  "start",
		"status":  "started",
	})
}

func (h *ServicePluginHandler) Stop(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	plugin := c.Param("plugin")

	c.JSON(http.StatusOK, gin.H{
		"node_id": nodeID,
		"plugin":  plugin,
		"action":  "stop",
		"status":  "stopped",
	})
}

func (h *ServicePluginHandler) Restart(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	plugin := c.Param("plugin")

	c.JSON(http.StatusOK, gin.H{
		"node_id": nodeID,
		"plugin":  plugin,
		"action":  "restart",
		"status":  "running",
	})
}

func (h *ServicePluginHandler) Pause(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	plugin := c.Param("plugin")

	c.JSON(http.StatusOK, gin.H{
		"node_id": nodeID,
		"plugin":  plugin,
		"action":  "pause",
		"status":  "paused",
	})
}

func (h *ServicePluginHandler) Status(c *gin.Context) {
	_, _ = strconv.ParseInt(c.Param("id"), 10, 64)
	plugin := c.Param("plugin")

	c.JSON(http.StatusOK, ServiceStatus{
		Plugin:   plugin,
		Status:   "running",
		Uptime:   3600,
	})
}

func (h *ServicePluginHandler) Config(c *gin.Context) {
	nodeID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	plugin := c.Param("plugin")

	var req struct {
		Config map[string]interface{} `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"node_id": nodeID,
		"plugin":  plugin,
		"config":  req.Config,
		"status":  "updated",
	})
}