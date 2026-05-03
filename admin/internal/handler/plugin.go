package handler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"anthill/admin/internal/model"
)

type PluginHandler struct {
	DB        *gorm.DB
	PluginDir string
}

func NewPluginHandler(db *gorm.DB, pluginDir string) *PluginHandler {
	os.MkdirAll(pluginDir, 0755)
	return &PluginHandler{DB: db, PluginDir: pluginDir}
}

func (h *PluginHandler) List(c *gin.Context) {
	plugins, err := model.ListPlugins(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, plugins)
}

func (h *PluginHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	name := c.PostForm("name")
	version := c.PostForm("version")
	pluginType := c.PostForm("type")
	description := c.PostForm("description")

	if name == "" || version == "" || pluginType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, version, type are required"})
		return
	}

	if pluginType != "service" && pluginType != "command" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'service' or 'command'"})
		return
	}

	filePath := filepath.Join(h.PluginDir, fmt.Sprintf("%s-%s.wasm", name, version))

	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer out.Close()

	hash := sha256.New()
	writer := io.MultiWriter(out, hash)

	if _, err := io.Copy(writer, file); err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	checksum := fmt.Sprintf("%x", hash.Sum(nil))

	plugin := &model.Plugin{
		Name:       name,
		Version:    version,
		Description: description,
		FilePath:   filePath,
		FileSize:   header.Size,
		PluginType: pluginType,
		Checksum:   checksum,
	}

	if err := h.DB.Create(plugin).Error; err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       plugin.ID,
		"checksum": checksum,
		"size":     header.Size,
	})
}

func (h *PluginHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	plugin, err := model.GetPluginByID(h.DB, id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := model.DeletePlugin(h.DB, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	os.Remove(plugin.FilePath)

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

func (h *PluginHandler) Download(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	plugin, err := model.GetPluginByID(h.DB, id)
	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.wasm", plugin.Name))
	c.File(plugin.FilePath)
}