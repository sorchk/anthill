package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type SessionHandler struct {
	DB        *gorm.DB
	JWTSecret []byte
}

func NewSessionHandler(db *gorm.DB, jwtSecret []byte) *SessionHandler {
	return &SessionHandler{DB: db, JWTSecret: jwtSecret}
}

func (h *SessionHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var sessions []model.Session
	err := h.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&sessions).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

func (h *SessionHandler) Revoke(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID, _ := c.Get("user_id")

	result := h.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Session{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session revoked"})
}

func (h *SessionHandler) RevokeAll(c *gin.Context) {
	userID, _ := c.Get("user_id")

	h.DB.Where("user_id = ?", userID).Delete(&model.Session{})

	c.JSON(http.StatusOK, gin.H{"message": "All sessions revoked"})
}

func (h *SessionHandler) CleanupExpired() error {
	return h.DB.Where("expires_at < ?", time.Now()).Delete(&model.Session{}).Error
}