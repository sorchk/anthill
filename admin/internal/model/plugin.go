package model

import (
	"time"

	"gorm.io/gorm"
)

type Plugin struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Version     string         `gorm:"size:50;not null" json:"version"`
	Description string         `gorm:"type:text" json:"description"`
	FilePath    string         `gorm:"size:512;not null" json:"file_path"`
	FileSize    int64          `json:"file_size"`
	PluginType  string         `gorm:"size:50;not null" json:"plugin_type"`
	Checksum    string         `gorm:"size:64" json:"checksum"`
	UploadedBy  int64          `json:"uploaded_by"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func ListPlugins(db *gorm.DB) ([]Plugin, error) {
	var plugins []Plugin
	err := db.Order("name, version DESC").Find(&plugins).Error
	return plugins, err
}

func GetPluginByID(db *gorm.DB, id int64) (*Plugin, error) {
	var plugin Plugin
	err := db.First(&plugin, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}
	return &plugin, err
}

func CreatePlugin(db *gorm.DB, plugin *Plugin) error {
	return db.Create(plugin).Error
}

func DeletePlugin(db *gorm.DB, id int64) error {
	result := db.Delete(&Plugin{}, id)
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}