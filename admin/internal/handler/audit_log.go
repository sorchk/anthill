package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"anthill/admin/internal/model"
)

type AuditLogHandler struct {
	DB *sql.DB
}

func NewAuditLogHandler(db *sql.DB) *AuditLogHandler {
	return &AuditLogHandler{DB: db}
}

func (h *AuditLogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize

	var total int
	h.DB.QueryRow("SELECT COUNT(*) FROM audit_logs").Scan(&total)

	rows, err := h.DB.Query(`
		SELECT id, user_id, username, action, resource, method, path, ip, status, details, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var log model.AuditLog
		var details sql.NullString

		err := rows.Scan(&log.ID, &log.UserID, &log.Username, &log.Action, &log.Resource,
			&log.Method, &log.Path, &log.IP, &log.Status, &details, &log.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if details.Valid {
			log.Details = details.String
		}

		logs = append(logs, log)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      logs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *AuditLogHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var log model.AuditLog
	var details sql.NullString

	err := h.DB.QueryRow(`
		SELECT id, user_id, username, action, resource, method, path, ip, status, details, created_at
		FROM audit_logs WHERE id = ?
	`, id).Scan(&log.ID, &log.UserID, &log.Username, &log.Action, &log.Resource,
		&log.Method, &log.Path, &log.IP, &log.Status, &details, &log.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Log not found"})
		return
	}

	if details.Valid {
		log.Details = details.String
	}

	c.JSON(http.StatusOK, log)
}