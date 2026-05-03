package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	DB *sql.DB
}

func NewStatsHandler(db *sql.DB) *StatsHandler {
	return &StatsHandler{DB: db}
}

type DashboardStats struct {
	TotalNodes     int `json:"total_nodes"`
	OnlineNodes    int `json:"online_nodes"`
	OfflineNodes   int `json:"offline_nodes"`
	TotalPlugins   int `json:"total_plugins"`
	TotalUsers     int `json:"total_users"`
	TotalAuditLogs int `json:"total_audit_logs"`
}

func (h *StatsHandler) Dashboard(c *gin.Context) {
	stats := DashboardStats{}

	h.DB.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&stats.TotalNodes)
	h.DB.QueryRow("SELECT COUNT(*) FROM nodes WHERE last_seen IS NOT NULL AND datetime(last_seen) > datetime('now', '-2 minutes')").Scan(&stats.OnlineNodes)
	h.DB.QueryRow("SELECT COUNT(*) FROM nodes WHERE last_seen IS NULL OR datetime(last_seen) <= datetime('now', '-2 minutes')").Scan(&stats.OfflineNodes)
	h.DB.QueryRow("SELECT COUNT(*) FROM plugins").Scan(&stats.TotalPlugins)
	h.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	h.DB.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&stats.TotalAuditLogs)

	c.JSON(http.StatusOK, stats)
}

func (h *StatsHandler) Health(c *gin.Context) {
	var dbOK bool
	h.DB.QueryRow("SELECT 1").Scan(&dbOK)

	c.JSON(http.StatusOK, gin.H{
		"status":   "healthy",
		"database": "connected",
	})
}