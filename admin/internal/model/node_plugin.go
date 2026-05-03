package model

import (
	"time"

	"gorm.io/gorm"
)

type NodePlugin struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	NodeID      int64          `gorm:"not null;uniqueIndex:idx_node_plugin" json:"node_id"`
	PluginName  string         `gorm:"size:255;not null;uniqueIndex:idx_node_plugin" json:"plugin_name"`
	Version     string         `gorm:"size:50;not null" json:"version"`
	PluginType  string         `gorm:"size:50;not null" json:"plugin_type"`
	Status      string         `gorm:"size:50;default:installing" json:"status"`
	Enabled     bool           `gorm:"default:true" json:"enabled"`
	InstalledAt time.Time     `gorm:"autoCreateTime" json:"installed_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type NodePluginInstall struct {
	PluginName string `json:"plugin_name" binding:"required"`
	Version    string `json:"version" binding:"required"`
}

func ListNodePlugins(db *gorm.DB, nodeID int64) ([]NodePlugin, error) {
	var plugins []NodePlugin
	err := db.Where("node_id = ?", nodeID).Find(&plugins).Error
	return plugins, err
}

func GetNodePlugin(db *gorm.DB, nodeID int64, pluginName string) (*NodePlugin, error) {
	var plugin NodePlugin
	err := db.Where("node_id = ? AND plugin_name = ?", nodeID, pluginName).First(&plugin).Error
	if err == gorm.ErrRecordNotFound {
		return nil, err
	}
	return &plugin, err
}

func CreateOrUpdateNodePlugin(db *gorm.DB, plugin *NodePlugin) error {
	existing, err := GetNodePlugin(db, plugin.NodeID, plugin.PluginName)
	if err == gorm.ErrRecordNotFound {
		return db.Create(plugin).Error
	}
	if err != nil {
		return err
	}
	plugin.ID = existing.ID
	return db.Save(plugin).Error
}

func UpdateNodePluginStatus(db *gorm.DB, nodeID int64, pluginName string, status string) error {
	return db.Model(&NodePlugin{}).Where("node_id = ? AND plugin_name = ?", nodeID, pluginName).Update("status", status).Error
}

func SetNodePluginEnabled(db *gorm.DB, nodeID int64, pluginName string, enabled bool) error {
	return db.Model(&NodePlugin{}).Where("node_id = ? AND plugin_name = ?", nodeID, pluginName).Update("enabled", enabled).Error
}

func DeleteNodePlugin(db *gorm.DB, nodeID int64, pluginName string) error {
	result := db.Where("node_id = ? AND plugin_name = ?", nodeID, pluginName).Delete(&NodePlugin{})
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}