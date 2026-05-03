package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type StatsHandler struct {
	DB *gorm.DB
}

func NewStatsHandler(db *gorm.DB) *StatsHandler {
	return &StatsHandler{DB: db}
}

type DashboardStats struct {
	TotalNodes     int64 `json:"total_nodes"`
	OnlineNodes    int64 `json:"online_nodes"`
	OfflineNodes   int64 `json:"offline_nodes"`
	TotalPlugins   int64 `json:"total_plugins"`
	TotalUsers     int64 `json:"total_users"`
	TotalAuditLogs int64 `json:"total_audit_logs"`
}

func (h *StatsHandler) Dashboard(c *gin.Context) {
	stats := DashboardStats{}

	h.DB.Model(&model.Node{}).Count(&stats.TotalNodes)
	h.DB.Model(&model.Plugin{}).Count(&stats.TotalPlugins)
	h.DB.Model(&model.User{}).Count(&stats.TotalUsers)
	h.DB.Model(&model.AuditLog{}).Count(&stats.TotalAuditLogs)

	threshold := time.Now().Add(-2 * time.Minute)
	h.DB.Model(&model.Node{}).Where("last_seen IS NOT NULL AND last_seen > ?", threshold).Count(&stats.OnlineNodes)
	h.DB.Model(&model.Node{}).Where("last_seen IS NULL OR last_seen <= ?", threshold).Count(&stats.OfflineNodes)

	c.JSON(http.StatusOK, stats)
}

func (h *StatsHandler) Health(c *gin.Context) {
	sqlDB, err := h.DB.DB()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "disconnected",
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"database": "disconnected",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"database": "connected",
	})
}