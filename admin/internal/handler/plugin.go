package handler

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	"anthill/admin/internal/model"
)

type PluginHandler struct {
	DB        *sql.DB
	PluginDir string
}

func NewPluginHandler(db *sql.DB, pluginDir string) *PluginHandler {
	os.MkdirAll(pluginDir, 0755)
	return &PluginHandler{DB: db, PluginDir: pluginDir}
}

func (h *PluginHandler) List(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT id, name, version, description, file_path, file_size, plugin_type, checksum, created_at
		FROM plugins ORDER BY name, version DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var plugins []model.Plugin
	for rows.Next() {
		var p model.Plugin
		err := rows.Scan(&p.ID, &p.Name, &p.Version, &p.Description, &p.FilePath, &p.FileSize, &p.PluginType, &p.Checksum, &p.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		plugins = append(plugins, p)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	checksum := fmt.Sprintf("%x", hash.Sum(nil))

	result, err := h.DB.Exec(`
		INSERT INTO plugins (name, version, description, file_path, file_size, plugin_type, checksum)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, name, version, description, filePath, header.Size, pluginType, checksum)

	if err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, _ := result.LastInsertId()

	c.JSON(http.StatusCreated, gin.H{
		"id":       id,
		"checksum": checksum,
		"size":     header.Size,
	})
}

func (h *PluginHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var filePath string
	err := h.DB.QueryRow("SELECT file_path FROM plugins WHERE id = ?", id).Scan(&filePath)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	_, err = h.DB.Exec("DELETE FROM plugins WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	os.Remove(filePath)

	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

func (h *PluginHandler) Download(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var filePath, name string
	err := h.DB.QueryRow("SELECT file_path, name FROM plugins WHERE id = ?", id).Scan(&filePath, &name)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.wasm", name))
	c.File(filePath)
}