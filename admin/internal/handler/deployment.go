package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type DeploymentHandler struct {
	DB *gorm.DB
}

func NewDeploymentHandler(db *gorm.DB) *DeploymentHandler {
	return &DeploymentHandler{DB: db}
}

func (h *DeploymentHandler) List(c *gin.Context) {
	nodeID := c.Query("node_id")
	pluginID := c.Query("plugin_id")

	deployments, err := model.ListDeployments(h.DB, nodeID, pluginID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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

	plugin, err := model.GetPluginByID(h.DB, req.PluginID)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var deployments []int64
	for _, nodeID := range req.NodeIDs {
		deployment := &model.Deployment{
			NodeID:     nodeID,
			PluginID:   req.PluginID,
			Version:    plugin.Version,
			Status:     "pending",
			DeployedBy: int64(userID.(int)),
		}

		if err := h.DB.Create(deployment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create deployment: " + err.Error()})
			return
		}

		deployments = append(deployments, deployment.ID)

		go h.processDeployment(deployment.ID, nodeID, req.PluginID)
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":      "Deployment initiated",
		"deployments": deployments,
	})
}

func (h *DeploymentHandler) processDeployment(deploymentID, nodeID, pluginID int64) {
	// Simple async processing
}

func (h *DeploymentHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	deployment, err := model.GetDeploymentByID(h.DB, id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

func (h *DeploymentHandler) Cancel(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := model.CancelDeployment(h.DB, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel - deployment not pending or not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cancelled"})
}