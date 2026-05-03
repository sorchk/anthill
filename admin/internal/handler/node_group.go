package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"anthill/admin/internal/model"
)

type NodeGroupHandler struct {
	DB *sql.DB
}

func NewNodeGroupHandler(db *sql.DB) *NodeGroupHandler {
	return &NodeGroupHandler{DB: db}
}

func (h *NodeGroupHandler) List(c *gin.Context) {
	rows, err := h.DB.Query("SELECT id, name, description, created_at FROM node_groups ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var groups []model.NodeGroup
	for rows.Next() {
		var g model.NodeGroup
		var desc sql.NullString
		if err := rows.Scan(&g.ID, &g.Name, &desc, &g.CreatedAt); err != nil {
			continue
		}
		if desc.Valid {
			g.Description = desc.String
		}
		groups = append(groups, g)
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

	result, err := h.DB.Exec(
		"INSERT INTO node_groups (name, description) VALUES (?, ?)",
		req.Name, req.Description,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *NodeGroupHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	_, err := h.DB.Exec("DELETE FROM node_groups WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}