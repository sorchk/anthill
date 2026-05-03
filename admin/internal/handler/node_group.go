package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type NodeGroupHandler struct {
	DB *gorm.DB
}

func NewNodeGroupHandler(db *gorm.DB) *NodeGroupHandler {
	return &NodeGroupHandler{DB: db}
}

func (h *NodeGroupHandler) List(c *gin.Context) {
	groups, err := model.ListNodeGroups(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (h *NodeGroupHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group := &model.NodeGroup{
		Name:        req.Name,
		Description: req.Description,
	}

	if err := h.DB.Create(group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": group.ID})
}

func (h *NodeGroupHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := model.DeleteNodeGroup(h.DB, id); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Node group not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}