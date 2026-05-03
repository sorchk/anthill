package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"anthill/admin/internal/middleware"
	"anthill/admin/internal/model"
)

type AuthHandler struct {
	DB *gorm.DB
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	var user model.User
	err := h.DB.Where("username = ?", req.Username).First(&user).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := middleware.GenerateToken(int(user.ID), user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	sessionToken := c.GetHeader("Authorization")
	if sessionToken != "" {
		sessionToken = strings.TrimPrefix(sessionToken, "Bearer ")
	}

	if sessionToken != "" {
		session := &model.Session{
			UserID:    user.ID,
			Token:     sessionToken,
			IP:        c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		h.DB.Create(session)
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	c.JSON(http.StatusOK, gin.H{
		"id":       userID,
		"username": username,
		"role":     role,
	})
}

func (h *AuthHandler) Check(c *gin.Context) {
	var count int64
	h.DB.Model(&model.User{}).Count(&count)
	c.JSON(http.StatusOK, gin.H{"initialized": count > 0})
}

func (h *AuthHandler) Init(c *gin.Context) {
	var count int64
	h.DB.Model(&model.User{}).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "System already initialized"})
		return
	}

	var req struct {
		Username        string `json:"username" binding:"required"`
		Password        string `json:"password" binding:"required,min=6"`
		ConfirmPassword string `json:"confirmPassword" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request"})
		return
	}

	if req.Password != req.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Passwords do not match"})
		return
	}

	user := &model.User{
		Username: req.Username,
		Role:     "admin",
	}
	if err := user.SetPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to hash password"})
		return
	}

	if err := h.DB.Create(user).Error; err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") || strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Username already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}